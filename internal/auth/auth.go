// Package auth stores and retrieves Bitbucket credentials. The canonical
// store is the OS keyring (via zalando/go-keyring). When the keyring is
// unavailable we fall back to a 0600 file under the config dir, but only
// when BT_ALLOW_INSECURE_STORE=1 is set.
//
// Env-var fallback is also supported so `bt` works in CI without interactive
// login: BT_TOKEN / BT_USERNAME / BT_APP_PASSWORD / BT_PAT.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arogan178/bitbucket-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	keyringService = "bt.bitbucket"

	envToken       = "BT_TOKEN"        // Cloud API token or DC PAT
	envUsername    = "BT_USERNAME"     // Cloud username (with BT_APP_PASSWORD)
	envAppPassword = "BT_APP_PASSWORD" // Cloud app password
	envPAT         = "BT_PAT"          // Explicit DC PAT
	envEmail       = "BT_EMAIL"        // Cloud email (with BT_TOKEN)
	envInsecure    = "BT_ALLOW_INSECURE_STORE"
)

// Credential captures what we need to authenticate one request. Exactly
// one of Token (bearer or basic via email) or (Username + AppPassword)
// is populated.
type Credential struct {
	Kind        config.Kind `json:"kind"`
	ContextName string      `json:"context"`
	// Principal is the email (Cloud with API token) or username (Cloud
	// app password / DC).
	Principal string `json:"principal,omitempty"`
	// Secret is the token, app password, or PAT.
	Secret string `json:"secret,omitempty"`
	// Mode: "api_token" | "app_password" | "pat"
	Mode string `json:"mode"`
}

// BasicAuth returns the (user, pass) tuple suitable for HTTP Basic.
func (c Credential) BasicAuth() (string, string) {
	return c.Principal, c.Secret
}

// ErrMissing indicates no credential is available for the context.
var ErrMissing = errors.New("no bitbucket credentials found (run `bt auth login` or set BT_TOKEN)")

// Load returns the credential for the given context, consulting env first,
// then the keyring, then the insecure file store if opted in.
func Load(ctx *config.Context) (Credential, error) {
	if cred, ok := fromEnv(ctx); ok {
		return cred, nil
	}
	if cred, ok, err := fromKeyring(ctx); err == nil && ok {
		return cred, nil
	}
	if cred, ok, err := fromInsecureStore(ctx); err == nil && ok {
		return cred, nil
	}
	return Credential{}, ErrMissing
}

// Store persists a credential for the given context.
func Store(ctx *config.Context, cred Credential) error {
	cred.ContextName = ctx.Name
	cred.Kind = ctx.Kind
	payload, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, ctx.Name, string(payload)); err != nil {
		if os.Getenv(envInsecure) == "1" {
			return writeInsecureStore(ctx.Name, payload)
		}
		return fmt.Errorf("keyring unavailable: %w (set %s=1 to use a file store)", err, envInsecure)
	}
	return nil
}

// Delete removes credentials for the context from all stores we control.
func Delete(ctx *config.Context) error {
	_ = keyring.Delete(keyringService, ctx.Name)
	path, err := insecurePath(ctx.Name)
	if err == nil {
		_ = os.Remove(path)
	}
	return nil
}

func fromEnv(ctx *config.Context) (Credential, bool) {
	switch ctx.Kind {
	case config.KindCloud:
		if email := os.Getenv(envEmail); email != "" {
			if tok := os.Getenv(envToken); tok != "" {
				return Credential{Principal: email, Secret: tok, Mode: "api_token"}, true
			}
		}
		if user := os.Getenv(envUsername); user != "" {
			if pw := os.Getenv(envAppPassword); pw != "" {
				return Credential{Principal: user, Secret: pw, Mode: "app_password"}, true
			}
		}
	case config.KindDataCenter:
		if pat := os.Getenv(envPAT); pat != "" {
			principal := os.Getenv(envUsername)
			if principal == "" {
				principal = ctx.Username
			}
			return Credential{Principal: principal, Secret: pat, Mode: "pat"}, true
		}
		if tok := os.Getenv(envToken); tok != "" {
			principal := os.Getenv(envUsername)
			if principal == "" {
				principal = ctx.Username
			}
			return Credential{Principal: principal, Secret: tok, Mode: "pat"}, true
		}
	}
	return Credential{}, false
}

func fromKeyring(ctx *config.Context) (Credential, bool, error) {
	raw, err := keyring.Get(keyringService, ctx.Name)
	if err != nil {
		return Credential{}, false, err
	}
	var c Credential
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return Credential{}, false, err
	}
	return c, true, nil
}

func fromInsecureStore(ctx *config.Context) (Credential, bool, error) {
	path, err := insecurePath(ctx.Name)
	if err != nil {
		return Credential{}, false, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Credential{}, false, nil
		}
		return Credential{}, false, err
	}
	var c Credential
	if err := json.Unmarshal(data, &c); err != nil {
		return Credential{}, false, err
	}
	return c, true, nil
}

func writeInsecureStore(name string, payload []byte) error {
	path, err := insecurePath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

func insecurePath(name string) (string, error) {
	cfgPath, err := config.Path()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(cfgPath)
	return filepath.Join(dir, "credentials", name+".json"), nil
}
