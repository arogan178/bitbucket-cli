package bitbucket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/arogan178/bitbucket-cli/internal/auth"
)

// httpClient is the shared HTTP plumbing used by Cloud and DC
// implementations. It sets Basic auth, JSON content negotiation,
// rate-limit retry on 429, and decodes errors.
type httpClient struct {
	baseURL string
	cred    auth.Credential
	http    *http.Client
}

func newHTTPClient(baseURL string, cred auth.Credential) *httpClient {
	return &httpClient{
		baseURL: baseURL,
		cred:    cred,
		http: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// APIError represents a non-2xx response.
type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("bitbucket API %d: %s", e.Status, e.Body)
}

// Is allows errors.Is(err, &APIError{Status: 404}).
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return t.Status == 0 || t.Status == e.Status
}

// doJSON executes a request and decodes the response into `out`. `body`
// can be nil or an arbitrary struct.
func (c *httpClient) doJSON(ctx context.Context, method, path string, params map[string]string, body any, out any) error {
	data, err := c.doRaw(ctx, method, path, params, body, "application/json")
	if err != nil {
		return err
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// doRaw performs the call and returns raw bytes.
func (c *httpClient) doRaw(ctx context.Context, method, path string, params map[string]string, body any, accept string) ([]byte, error) {
	res, err := c.do(ctx, method, path, params, body, accept)
	if err != nil {
		return nil, err
	}
	defer res.Close()
	return io.ReadAll(res)
}

// doStream returns the raw body (caller must close). Used for diffs and logs.
func (c *httpClient) doStream(ctx context.Context, method, path string, params map[string]string, body any, accept string) (io.ReadCloser, error) {
	return c.do(ctx, method, path, params, body, accept)
}

func (c *httpClient) do(ctx context.Context, method, path string, params map[string]string, body any, accept string) (io.ReadCloser, error) {
	if accept == "" {
		accept = "application/json"
	}

	u, err := c.buildURL(path, params)
	if err != nil {
		return nil, err
	}

	var reader io.Reader
	if body != nil {
		switch b := body.(type) {
		case io.Reader:
			reader = b
		case []byte:
			reader = bytes.NewReader(b)
		default:
			data, err := json.Marshal(b)
			if err != nil {
				return nil, err
			}
			reader = bytes.NewReader(data)
		}
	}

	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, reader)
		if err != nil {
			return nil, err
		}
		if principal, secret := c.cred.BasicAuth(); principal != "" {
			req.SetBasicAuth(principal, secret)
		} else if c.cred.Secret != "" {
			req.Header.Set("Authorization", "Bearer "+c.cred.Secret)
		}
		req.Header.Set("Accept", accept)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		if res.StatusCode == 429 && attempt < maxRetries-1 {
			wait := retryAfter(res)
			res.Body.Close()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
			// Reset body reader for the retry.
			if body != nil {
				if data, ok := body.([]byte); ok {
					reader = bytes.NewReader(data)
				} else if _, isReader := body.(io.Reader); !isReader {
					data, _ := json.Marshal(body)
					reader = bytes.NewReader(data)
				}
			}
			continue
		}

		if res.StatusCode >= 400 {
			raw, _ := io.ReadAll(res.Body)
			res.Body.Close()
			return nil, &APIError{Status: res.StatusCode, Body: string(raw)}
		}

		return res.Body, nil
	}
	return nil, errors.New("exceeded retry budget")
}

func (c *httpClient) buildURL(path string, params map[string]string) (string, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	// `path` may be absolute (starts with /) or relative.
	if len(path) > 0 && path[0] == '/' {
		u.Path = path
	} else {
		u.Path = u.Path + "/" + path
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func retryAfter(res *http.Response) time.Duration {
	if v := res.Header.Get("Retry-After"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return time.Duration(n) * time.Second
		}
	}
	return 2 * time.Second
}
