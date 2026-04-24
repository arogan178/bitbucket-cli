package bitbucket

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/config"
)

// dcClient targets a Bitbucket Data Center host at /rest/api/1.0.
type dcClient struct {
	ctx  *config.Context
	http *httpClient
}

func newDataCenterClient(ctx *config.Context, cred auth.Credential) (Client, error) {
	if ctx.Host == "" {
		return nil, fmt.Errorf("data center context %q has no host", ctx.Name)
	}
	return &dcClient{
		ctx:  ctx,
		http: newHTTPClient(ctx.Host, cred),
	}, nil
}

func (c *dcClient) Kind() config.Kind { return config.KindDataCenter }
func (c *dcClient) Host() string      { return c.ctx.Host }

func (c *dcClient) Repos() RepoService               { return &dcRepos{c} }
func (c *dcClient) PullRequests() PullRequestService { return &dcPRs{c} }
func (c *dcClient) Branches() BranchService          { return &dcBranches{c} }
func (c *dcClient) Compare() CompareService          { return &dcCompare{c} }
func (c *dcClient) Pipelines() PipelineService       { return &dcPipelines{c} }
func (c *dcClient) Issues() IssueService             { return &dcIssues{c} }
func (c *dcClient) Webhooks() WebhookService         { return &dcWebhooks{c} }
func (c *dcClient) Raw() RawService                  { return &rawDC{c} }

func (c *dcClient) project() string { return c.ctx.Project }

type dcPage[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

// ---------- repos ----------

type dcRepos struct{ c *dcClient }

type dcRepo struct {
	ID            int    `json:"id"`
	Slug          string `json:"slug"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Public        bool   `json:"public"`
	DefaultBranch struct{ DisplayID string `json:"displayId"` } `json:"defaultBranch"`
	Project       struct{ Key, Name string } `json:"project"`
	Links         struct {
		Self  []struct{ Href string } `json:"self"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

func (r *dcRepo) toRepo() Repo {
	var https, ssh, web string
	for _, c := range r.Links.Clone {
		switch c.Name {
		case "http", "https":
			https = c.Href
		case "ssh":
			ssh = c.Href
		}
	}
	if len(r.Links.Self) > 0 {
		web = r.Links.Self[0].Href
	}
	return Repo{
		Slug:          r.Slug,
		Name:          r.Name,
		Description:   r.Description,
		Project:       r.Project.Key,
		Private:       !r.Public,
		DefaultBranch: r.DefaultBranch.DisplayID,
		HTTPSCloneURL: https,
		SSHCloneURL:   ssh,
		WebURL:        web,
	}
}

func (s *dcRepos) List(ctx context.Context, opts ListOptions) ([]Repo, error) {
	proj := s.c.project()
	if proj == "" {
		return nil, fmt.Errorf("data center repo list requires a project")
	}
	params := map[string]string{}
	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["filter"] = opts.Query
	}
	var page dcPage[dcRepo]
	if err := s.c.http.doJSON(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos", proj), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Repo, len(page.Values))
	for i, r := range page.Values {
		out[i] = r.toRepo()
	}
	return out, nil
}

func (s *dcRepos) Get(ctx context.Context, slug string) (*Repo, error) {
	var r dcRepo
	if err := s.c.http.doJSON(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s", s.c.project(), slug), nil, nil, &r); err != nil {
		return nil, err
	}
	out := r.toRepo()
	return &out, nil
}

func (s *dcRepos) Create(ctx context.Context, slug, description string, private bool) (*Repo, error) {
	body := map[string]any{
		"name":        slug,
		"scmId":       "git",
		"description": description,
		"public":      !private,
	}
	var r dcRepo
	if err := s.c.http.doJSON(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos", s.c.project()), nil, body, &r); err != nil {
		return nil, err
	}
	out := r.toRepo()
	return &out, nil
}

func (s *dcRepos) Delete(ctx context.Context, slug string) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s", s.c.project(), slug), nil, nil, "application/json")
	return err
}

// ---------- pull requests ----------

type dcPRs struct{ c *dcClient }

type dcPR struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Author      struct {
		User struct{ DisplayName, Name string } `json:"user"`
	} `json:"author"`
	Reviewers []struct {
		User     struct{ DisplayName, Name string } `json:"user"`
		Approved bool                              `json:"approved"`
	} `json:"reviewers"`
	FromRef struct {
		DisplayID string `json:"displayId"`
	} `json:"fromRef"`
	ToRef struct {
		DisplayID string `json:"displayId"`
	} `json:"toRef"`
	Links struct {
		Self []struct{ Href string } `json:"self"`
	} `json:"links"`
	CreatedDate int64 `json:"createdDate"`
	UpdatedDate int64 `json:"updatedDate"`
}

func (p *dcPR) toPR() PullRequest {
	authorName := p.Author.User.DisplayName
	if authorName == "" {
		authorName = p.Author.User.Name
	}
	reviewers := make([]string, 0, len(p.Reviewers))
	approvals := make([]string, 0)
	for _, r := range p.Reviewers {
		name := r.User.DisplayName
		if name == "" {
			name = r.User.Name
		}
		if name == "" {
			continue
		}
		reviewers = append(reviewers, name)
		if r.Approved {
			approvals = append(approvals, name)
		}
	}
	var web string
	if len(p.Links.Self) > 0 {
		web = p.Links.Self[0].Href
	}
	return PullRequest{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		State:       p.State,
		Author:      authorName,
		Source:      p.FromRef.DisplayID,
		Target:      p.ToRef.DisplayID,
		WebURL:      web,
		CreatedAt:   time.UnixMilli(p.CreatedDate),
		UpdatedAt:   time.UnixMilli(p.UpdatedDate),
		Reviewers:   reviewers,
		Approvals:   approvals,
	}
}

func (s *dcPRs) path(slug string, id int) string {
	if id == 0 {
		return fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests", s.c.project(), slug)
	}
	return fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", s.c.project(), slug, id)
}

func (s *dcPRs) List(ctx context.Context, slug string, opts ListOptions) ([]PullRequest, error) {
	params := map[string]string{"state": "OPEN"}
	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["filterText"] = opts.Query
	}
	var page dcPage[dcPR]
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, 0), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]PullRequest, len(page.Values))
	for i, pr := range page.Values {
		out[i] = pr.toPR()
	}
	return out, nil
}

func (s *dcPRs) Get(ctx context.Context, slug string, id int) (*PullRequest, error) {
	var pr dcPR
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id), nil, nil, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *dcPRs) Create(ctx context.Context, slug string, in CreatePullRequestInput) (*PullRequest, error) {
	revs := make([]map[string]any, 0, len(in.Reviewers))
	for _, r := range in.Reviewers {
		revs = append(revs, map[string]any{"user": map[string]string{"name": r}})
	}
	body := map[string]any{
		"title":       in.Title,
		"description": in.Description,
		"state":       "OPEN",
		"open":        true,
		"closed":      false,
		"fromRef": map[string]any{
			"id":         "refs/heads/" + in.Source,
			"repository": map[string]any{"slug": slug, "project": map[string]string{"key": s.c.project()}},
		},
		"toRef": map[string]any{
			"id":         "refs/heads/" + in.Target,
			"repository": map[string]any{"slug": slug, "project": map[string]string{"key": s.c.project()}},
		},
		"reviewers": revs,
	}
	var pr dcPR
	if err := s.c.http.doJSON(ctx, "POST", s.path(slug, 0), nil, body, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *dcPRs) Edit(ctx context.Context, slug string, id int, in EditPullRequestInput) (*PullRequest, error) {
	current, err := s.Get(ctx, slug, id)
	if err != nil {
		return nil, err
	}
	var rawVersion int
	var v dcPR
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id), nil, nil, &v); err == nil {
		rawVersion = v.Version
	}
	body := map[string]any{
		"id":      id,
		"version": rawVersion,
	}
	if in.Title != nil {
		body["title"] = *in.Title
	} else {
		body["title"] = current.Title
	}
	if in.Description != nil {
		body["description"] = *in.Description
	}
	if in.Target != nil {
		body["toRef"] = map[string]any{
			"id":         "refs/heads/" + *in.Target,
			"repository": map[string]any{"slug": slug, "project": map[string]string{"key": s.c.project()}},
		}
	}
	var pr dcPR
	if err := s.c.http.doJSON(ctx, "PUT", s.path(slug, id), nil, body, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *dcPRs) Merge(ctx context.Context, slug string, id int, opts MergeOptions) error {
	// DC requires the current PR version.
	var v dcPR
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id), nil, nil, &v); err != nil {
		return err
	}
	_, err := s.c.http.doRaw(ctx, "POST", s.path(slug, id)+"/merge", map[string]string{"version": strconv.Itoa(v.Version)}, map[string]any{}, "application/json")
	return err
}

func (s *dcPRs) Decline(ctx context.Context, slug string, id int) error {
	var v dcPR
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id), nil, nil, &v); err != nil {
		return err
	}
	_, err := s.c.http.doRaw(ctx, "POST", s.path(slug, id)+"/decline", map[string]string{"version": strconv.Itoa(v.Version)}, map[string]any{}, "application/json")
	return err
}

func (s *dcPRs) Approve(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "POST", s.path(slug, id)+"/approve", nil, map[string]any{}, "application/json")
	return err
}

func (s *dcPRs) Unapprove(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", s.path(slug, id)+"/approve", nil, nil, "application/json")
	return err
}

func (s *dcPRs) Comment(ctx context.Context, slug string, id int, body string) (*Comment, error) {
	var res struct {
		ID          int   `json:"id"`
		Text        string `json:"text"`
		CreatedDate int64  `json:"createdDate"`
		Author      struct{ DisplayName string `json:"displayName"` } `json:"author"`
	}
	payload := map[string]any{"text": body}
	if err := s.c.http.doJSON(ctx, "POST", s.path(slug, id)+"/comments", nil, payload, &res); err != nil {
		return nil, err
	}
	return &Comment{ID: res.ID, Body: res.Text, Author: res.Author.DisplayName, CreatedAt: time.UnixMilli(res.CreatedDate)}, nil
}

func (s *dcPRs) Comments(ctx context.Context, slug string, id int) ([]Comment, error) {
	var page dcPage[struct {
		ID          int   `json:"id"`
		Text        string `json:"text"`
		CreatedDate int64  `json:"createdDate"`
		Author      struct{ DisplayName string `json:"displayName"` } `json:"author"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id)+"/activities", nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Comment, 0)
	for _, v := range page.Values {
		if v.Text == "" {
			continue
		}
		out = append(out, Comment{ID: v.ID, Body: v.Text, Author: v.Author.DisplayName, CreatedAt: time.UnixMilli(v.CreatedDate)})
	}
	return out, nil
}

func (s *dcPRs) Checks(ctx context.Context, slug string, id int) ([]BuildStatus, error) {
	// Fetch commits on the PR, then the build statuses on its head.
	var commits dcPage[struct {
		ID string `json:"id"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug, id)+"/commits", map[string]string{"limit": "1"}, nil, &commits); err != nil {
		return nil, err
	}
	if len(commits.Values) == 0 {
		return nil, nil
	}
	head := commits.Values[0].ID
	var page dcPage[struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		State       string `json:"state"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", "/rest/build-status/1.0/commits/"+head, nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]BuildStatus, 0, len(page.Values))
	for _, v := range page.Values {
		out = append(out, BuildStatus{Key: v.Key, Name: v.Name, State: v.State, URL: v.URL, Description: v.Description})
	}
	return out, nil
}

func (s *dcPRs) Diff(ctx context.Context, slug string, id int) (io.ReadCloser, error) {
	return s.c.http.doStream(ctx, "GET", s.path(slug, id)+".diff", nil, nil, "text/plain")
}

// ---------- branches ----------

type dcBranches struct{ c *dcClient }

func (s *dcBranches) base(slug string) string {
	return fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/branches", s.c.project(), slug)
}

func (s *dcBranches) List(ctx context.Context, slug string, opts ListOptions) ([]Branch, error) {
	params := map[string]string{}
	if opts.Limit > 0 {
		params["limit"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["filterText"] = opts.Query
	}
	var page dcPage[struct {
		ID          string `json:"id"`
		DisplayID   string `json:"displayId"`
		LatestCommit string `json:"latestCommit"`
		IsDefault   bool   `json:"isDefault"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Branch, 0, len(page.Values))
	for _, b := range page.Values {
		out = append(out, Branch{Name: b.DisplayID, Target: b.LatestCommit, IsDefault: b.IsDefault})
	}
	return out, nil
}

func (s *dcBranches) Create(ctx context.Context, slug, name, from string) (*Branch, error) {
	body := map[string]any{
		"name":       name,
		"startPoint": from,
	}
	var res struct {
		DisplayID    string `json:"displayId"`
		LatestCommit string `json:"latestCommit"`
	}
	if err := s.c.http.doJSON(ctx, "POST", s.base(slug), nil, body, &res); err != nil {
		return nil, err
	}
	return &Branch{Name: res.DisplayID, Target: res.LatestCommit}, nil
}

func (s *dcBranches) Delete(ctx context.Context, slug, name string) error {
	body := map[string]any{"name": "refs/heads/" + name}
	_, err := s.c.http.doRaw(ctx, "DELETE", fmt.Sprintf("/rest/branch-utils/1.0/projects/%s/repos/%s/branches", s.c.project(), slug), nil, body, "application/json")
	return err
}

func (s *dcBranches) SetDefault(ctx context.Context, slug, name string) error {
	body := map[string]any{"id": "refs/heads/" + name}
	_, err := s.c.http.doRaw(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-branch", s.c.project(), slug), nil, body, "application/json")
	return err
}

// ---------- compare ----------

type dcCompare struct{ c *dcClient }

func (s *dcCompare) base(slug string) string {
	return fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s", s.c.project(), slug)
}

// Data Center exposes /compare/diff and /compare/commits. `from` maps to
// the source ref, `to` to the target ref; three-dot / two-dot is
// controlled by `?withComments=false&whitespace=ignore-all` style flags
// on /commits. We translate naturally.
func (s *dcCompare) Diff(ctx context.Context, slug, from, to string, threeDot bool) (io.ReadCloser, error) {
	params := map[string]string{
		"from": to,
		"to":   from,
	}
	// Bitbucket DC returns unified diff at /compare/diff.
	return s.c.http.doStream(ctx, "GET", s.base(slug)+"/compare/diff", params, nil, "text/plain")
}

func (s *dcCompare) Commits(ctx context.Context, slug, from, to string) ([]Commit, error) {
	params := map[string]string{
		"from": to,
		"to":   from,
	}
	var page dcPage[struct {
		ID            string `json:"id"`
		DisplayID     string `json:"displayId"`
		Message       string `json:"message"`
		AuthorTimestamp int64 `json:"authorTimestamp"`
		Author        struct {
			DisplayName string `json:"displayName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"author"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug)+"/compare/commits", params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Commit, 0, len(page.Values))
	for _, v := range page.Values {
		out = append(out, Commit{
			SHA:       v.ID,
			ShortSHA:  v.DisplayID,
			Author:    v.Author.DisplayName,
			Email:     v.Author.EmailAddress,
			Message:   v.Message,
			CreatedAt: time.UnixMilli(v.AuthorTimestamp),
		})
	}
	return out, nil
}

// ---------- pipelines, issues, webhooks: not supported / minimal ----------

type dcPipelines struct{ c *dcClient }

func (s *dcPipelines) List(ctx context.Context, slug string, opts ListOptions) ([]Pipeline, error) {
	return nil, fmt.Errorf("pipelines are a Bitbucket Cloud feature; use your DC CI system (Jenkins/Bamboo) directly")
}
func (s *dcPipelines) Get(ctx context.Context, slug, uuid string) (*Pipeline, error) {
	return nil, fmt.Errorf("pipelines are a Bitbucket Cloud feature")
}
func (s *dcPipelines) Run(ctx context.Context, slug, ref string, vars map[string]string) (*Pipeline, error) {
	return nil, fmt.Errorf("pipelines are a Bitbucket Cloud feature")
}
func (s *dcPipelines) Cancel(ctx context.Context, slug, uuid string) error {
	return fmt.Errorf("pipelines are a Bitbucket Cloud feature")
}
func (s *dcPipelines) Logs(ctx context.Context, slug, uuid string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("pipelines are a Bitbucket Cloud feature")
}

type dcIssues struct{ c *dcClient }

func (s *dcIssues) List(ctx context.Context, slug string, opts ListOptions) ([]Issue, error) {
	return nil, fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI; use Jira")
}
func (s *dcIssues) Get(ctx context.Context, slug string, id int) (*Issue, error) {
	return nil, fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI")
}
func (s *dcIssues) Create(ctx context.Context, slug, title, body, kind, priority string) (*Issue, error) {
	return nil, fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI")
}
func (s *dcIssues) Close(ctx context.Context, slug string, id int) error {
	return fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI")
}
func (s *dcIssues) Reopen(ctx context.Context, slug string, id int) error {
	return fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI")
}
func (s *dcIssues) Comment(ctx context.Context, slug string, id int, body string) (*Comment, error) {
	return nil, fmt.Errorf("Bitbucket Data Center does not expose issues through the CLI")
}

type dcWebhooks struct{ c *dcClient }

func (s *dcWebhooks) base(slug string) string {
	return fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks", s.c.project(), slug)
}

func (s *dcWebhooks) List(ctx context.Context, slug string) ([]Webhook, error) {
	var page dcPage[struct {
		ID     int      `json:"id"`
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug), nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Webhook, 0, len(page.Values))
	for _, w := range page.Values {
		out = append(out, Webhook{ID: strconv.Itoa(w.ID), URL: w.URL, Name: w.Name, Events: w.Events, Active: w.Active})
	}
	return out, nil
}

func (s *dcWebhooks) Create(ctx context.Context, slug, name, url string, events []string) (*Webhook, error) {
	body := map[string]any{
		"name":   name,
		"url":    url,
		"active": true,
		"events": events,
	}
	var w struct {
		ID     int      `json:"id"`
		Name   string   `json:"name"`
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	}
	if err := s.c.http.doJSON(ctx, "POST", s.base(slug), nil, body, &w); err != nil {
		return nil, err
	}
	return &Webhook{ID: strconv.Itoa(w.ID), URL: w.URL, Name: w.Name, Events: w.Events, Active: w.Active}, nil
}

func (s *dcWebhooks) Delete(ctx context.Context, slug, id string) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", s.base(slug)+"/"+id, nil, nil, "application/json")
	return err
}

type rawDC struct{ c *dcClient }

func (r *rawDC) Do(ctx context.Context, method, path string, params map[string]string, body io.Reader) ([]byte, error) {
	return r.c.http.doRaw(ctx, method, path, params, body, "application/json")
}
