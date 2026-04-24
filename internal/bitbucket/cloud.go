package bitbucket

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/config"
)

// cloudClient targets https://api.bitbucket.org/2.0.
type cloudClient struct {
	ctx  *config.Context
	http *httpClient
}

func newCloudClient(ctx *config.Context, cred auth.Credential) (Client, error) {
	host := ctx.Host
	if host == "" {
		host = "https://bitbucket.org"
	}
	// API host differs from web host.
	if strings.HasPrefix(host, "https://bitbucket.org") {
		host = "https://api.bitbucket.org"
	}
	return &cloudClient{
		ctx:  ctx,
		http: newHTTPClient(host, cred),
	}, nil
}

func (c *cloudClient) Kind() config.Kind { return config.KindCloud }
func (c *cloudClient) Host() string      { return c.http.baseURL }

func (c *cloudClient) Repos() RepoService               { return &cloudRepos{c} }
func (c *cloudClient) PullRequests() PullRequestService { return &cloudPRs{c} }
func (c *cloudClient) Branches() BranchService          { return &cloudBranches{c} }
func (c *cloudClient) Compare() CompareService          { return &cloudCompare{c} }
func (c *cloudClient) Pipelines() PipelineService       { return &cloudPipelines{c} }
func (c *cloudClient) Issues() IssueService             { return &cloudIssues{c} }
func (c *cloudClient) Webhooks() WebhookService         { return &cloudWebhooks{c} }
func (c *cloudClient) Raw() RawService                  { return &rawCloud{c} }

func (c *cloudClient) workspace() string {
	if c.ctx.Workspace != "" {
		return c.ctx.Workspace
	}
	return ""
}

// ---------- shared envelopes ----------

type cloudPage[T any] struct {
	Values []T    `json:"values"`
	Next   string `json:"next,omitempty"`
	Page   int    `json:"page,omitempty"`
}

// ---------- repos ----------

type cloudRepos struct{ c *cloudClient }

type cloudRepo struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Mainbranch  struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
	Links struct {
		HTML  struct{ Href string } `json:"html"`
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
	UpdatedOn time.Time `json:"updated_on"`
	Workspace struct {
		Slug string `json:"slug"`
	} `json:"workspace"`
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
}

func (r *cloudRepo) toRepo() Repo {
	var https, ssh string
	for _, c := range r.Links.Clone {
		switch c.Name {
		case "https":
			https = c.Href
		case "ssh":
			ssh = c.Href
		}
	}
	return Repo{
		Slug:          r.Slug,
		Name:          r.Name,
		Description:   r.Description,
		Private:       r.IsPrivate,
		DefaultBranch: r.Mainbranch.Name,
		HTTPSCloneURL: https,
		SSHCloneURL:   ssh,
		WebURL:        r.Links.HTML.Href,
		UpdatedAt:     r.UpdatedOn,
		Workspace:     r.Workspace.Slug,
		Project:       r.Project.Key,
	}
}

func (s *cloudRepos) List(ctx context.Context, opts ListOptions) ([]Repo, error) {
	ws := s.c.workspace()
	if ws == "" {
		return nil, fmt.Errorf("cloud repo list requires a workspace")
	}
	params := map[string]string{}
	if opts.Limit > 0 {
		params["pagelen"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["q"] = opts.Query
	}
	var page cloudPage[cloudRepo]
	if err := s.c.http.doJSON(ctx, "GET", "/2.0/repositories/"+ws, params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Repo, len(page.Values))
	for i, r := range page.Values {
		out[i] = r.toRepo()
	}
	return out, nil
}

func (s *cloudRepos) Get(ctx context.Context, slug string) (*Repo, error) {
	var r cloudRepo
	if err := s.c.http.doJSON(ctx, "GET", fmt.Sprintf("/2.0/repositories/%s/%s", s.c.workspace(), slug), nil, nil, &r); err != nil {
		return nil, err
	}
	out := r.toRepo()
	return &out, nil
}

func (s *cloudRepos) Create(ctx context.Context, slug, description string, private bool) (*Repo, error) {
	body := map[string]any{
		"scm":         "git",
		"description": description,
		"is_private":  private,
	}
	var r cloudRepo
	if err := s.c.http.doJSON(ctx, "POST", fmt.Sprintf("/2.0/repositories/%s/%s", s.c.workspace(), slug), nil, body, &r); err != nil {
		return nil, err
	}
	out := r.toRepo()
	return &out, nil
}

func (s *cloudRepos) Delete(ctx context.Context, slug string) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", fmt.Sprintf("/2.0/repositories/%s/%s", s.c.workspace(), slug), nil, nil, "application/json")
	return err
}

// ---------- pull requests ----------

type cloudPRs struct{ c *cloudClient }

type cloudPR struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Draft       bool   `json:"draft"`
	Author      struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	} `json:"author"`
	Source struct {
		Branch struct{ Name string } `json:"branch"`
	} `json:"source"`
	Destination struct {
		Branch struct{ Name string } `json:"branch"`
	} `json:"destination"`
	Links struct {
		HTML struct{ Href string } `json:"html"`
		Diff struct{ Href string } `json:"diff"`
	} `json:"links"`
	Reviewers []struct {
		DisplayName string `json:"display_name"`
		Nickname    string `json:"nickname"`
	} `json:"reviewers"`
	Participants []struct {
		User struct {
			DisplayName string `json:"display_name"`
			Nickname    string `json:"nickname"`
		} `json:"user"`
		Approved bool   `json:"approved"`
		Role     string `json:"role"`
	} `json:"participants"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

func (p *cloudPR) toPR() PullRequest {
	author := p.Author.DisplayName
	if author == "" {
		author = p.Author.Nickname
	}
	reviewers := make([]string, 0, len(p.Reviewers))
	for _, r := range p.Reviewers {
		name := r.DisplayName
		if name == "" {
			name = r.Nickname
		}
		if name != "" {
			reviewers = append(reviewers, name)
		}
	}
	approvals := make([]string, 0)
	for _, pt := range p.Participants {
		if !pt.Approved {
			continue
		}
		name := pt.User.DisplayName
		if name == "" {
			name = pt.User.Nickname
		}
		if name != "" {
			approvals = append(approvals, name)
		}
	}
	return PullRequest{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		State:       p.State,
		Author:      author,
		Source:      p.Source.Branch.Name,
		Target:      p.Destination.Branch.Name,
		WebURL:      p.Links.HTML.Href,
		CreatedAt:   p.CreatedOn,
		UpdatedAt:   p.UpdatedOn,
		Reviewers:   reviewers,
		Approvals:   approvals,
		Draft:       p.Draft,
	}
}

func (s *cloudPRs) prPath(slug string, id int) string {
	if id == 0 {
		return fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests", s.c.workspace(), slug)
	}
	return fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d", s.c.workspace(), slug, id)
}

func (s *cloudPRs) List(ctx context.Context, slug string, opts ListOptions) ([]PullRequest, error) {
	params := map[string]string{}
	if opts.Limit > 0 {
		params["pagelen"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["q"] = opts.Query
	}
	var page cloudPage[cloudPR]
	if err := s.c.http.doJSON(ctx, "GET", s.prPath(slug, 0), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]PullRequest, len(page.Values))
	for i, pr := range page.Values {
		out[i] = pr.toPR()
	}
	return out, nil
}

func (s *cloudPRs) Get(ctx context.Context, slug string, id int) (*PullRequest, error) {
	var pr cloudPR
	if err := s.c.http.doJSON(ctx, "GET", s.prPath(slug, id), nil, nil, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *cloudPRs) Create(ctx context.Context, slug string, in CreatePullRequestInput) (*PullRequest, error) {
	revs := make([]map[string]string, 0, len(in.Reviewers))
	for _, r := range in.Reviewers {
		revs = append(revs, map[string]string{"username": r})
	}
	body := map[string]any{
		"title":               in.Title,
		"description":         in.Description,
		"source":              map[string]any{"branch": map[string]string{"name": in.Source}},
		"destination":         map[string]any{"branch": map[string]string{"name": in.Target}},
		"close_source_branch": in.CloseSource,
	}
	if len(revs) > 0 {
		body["reviewers"] = revs
	}
	if in.Draft {
		body["draft"] = true
	}
	var pr cloudPR
	if err := s.c.http.doJSON(ctx, "POST", s.prPath(slug, 0), nil, body, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *cloudPRs) Edit(ctx context.Context, slug string, id int, in EditPullRequestInput) (*PullRequest, error) {
	body := map[string]any{}
	if in.Title != nil {
		body["title"] = *in.Title
	}
	if in.Description != nil {
		body["description"] = *in.Description
	}
	if in.Target != nil {
		body["destination"] = map[string]any{"branch": map[string]string{"name": *in.Target}}
	}
	var pr cloudPR
	if err := s.c.http.doJSON(ctx, "PUT", s.prPath(slug, id), nil, body, &pr); err != nil {
		return nil, err
	}
	out := pr.toPR()
	return &out, nil
}

func (s *cloudPRs) Merge(ctx context.Context, slug string, id int, opts MergeOptions) error {
	body := map[string]any{}
	if opts.Strategy != "" {
		body["merge_strategy"] = opts.Strategy
	}
	if opts.Message != "" {
		body["message"] = opts.Message
	}
	if opts.CloseSource {
		body["close_source_branch"] = true
	}
	_, err := s.c.http.doRaw(ctx, "POST", s.prPath(slug, id)+"/merge", nil, body, "application/json")
	return err
}

func (s *cloudPRs) Decline(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "POST", s.prPath(slug, id)+"/decline", nil, map[string]any{}, "application/json")
	return err
}

func (s *cloudPRs) Approve(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "POST", s.prPath(slug, id)+"/approve", nil, map[string]any{}, "application/json")
	return err
}

func (s *cloudPRs) Unapprove(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", s.prPath(slug, id)+"/approve", nil, nil, "application/json")
	return err
}

func (s *cloudPRs) Comment(ctx context.Context, slug string, id int, body string) (*Comment, error) {
	payload := map[string]any{"content": map[string]string{"raw": body}}
	var res struct {
		ID        int       `json:"id"`
		CreatedOn time.Time `json:"created_on"`
		User      struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	}
	if err := s.c.http.doJSON(ctx, "POST", s.prPath(slug, id)+"/comments", nil, payload, &res); err != nil {
		return nil, err
	}
	return &Comment{ID: res.ID, Author: res.User.DisplayName, Body: res.Content.Raw, CreatedAt: res.CreatedOn}, nil
}

func (s *cloudPRs) Comments(ctx context.Context, slug string, id int) ([]Comment, error) {
	var page cloudPage[struct {
		ID        int       `json:"id"`
		CreatedOn time.Time `json:"created_on"`
		User      struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.prPath(slug, id)+"/comments", nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Comment, 0, len(page.Values))
	for _, v := range page.Values {
		out = append(out, Comment{ID: v.ID, Author: v.User.DisplayName, Body: v.Content.Raw, CreatedAt: v.CreatedOn})
	}
	return out, nil
}

func (s *cloudPRs) Checks(ctx context.Context, slug string, id int) ([]BuildStatus, error) {
	var page cloudPage[struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		State       string `json:"state"`
		URL         string `json:"url"`
		Description string `json:"description"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.prPath(slug, id)+"/statuses", nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]BuildStatus, 0, len(page.Values))
	for _, v := range page.Values {
		out = append(out, BuildStatus{Key: v.Key, Name: v.Name, State: v.State, URL: v.URL, Description: v.Description})
	}
	return out, nil
}

func (s *cloudPRs) Diff(ctx context.Context, slug string, id int) (io.ReadCloser, error) {
	return s.c.http.doStream(ctx, "GET", s.prPath(slug, id)+"/diff", nil, nil, "text/plain")
}

// ---------- branches ----------

type cloudBranches struct{ c *cloudClient }

func (s *cloudBranches) path(slug string) string {
	return fmt.Sprintf("/2.0/repositories/%s/%s/refs/branches", s.c.workspace(), slug)
}

func (s *cloudBranches) List(ctx context.Context, slug string, opts ListOptions) ([]Branch, error) {
	params := map[string]string{}
	if opts.Limit > 0 {
		params["pagelen"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["q"] = opts.Query
	}
	var page cloudPage[struct {
		Name   string `json:"name"`
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.path(slug), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Branch, 0, len(page.Values))
	for _, b := range page.Values {
		out = append(out, Branch{Name: b.Name, Target: b.Target.Hash})
	}
	return out, nil
}

func (s *cloudBranches) Create(ctx context.Context, slug, name, from string) (*Branch, error) {
	body := map[string]any{
		"name":   name,
		"target": map[string]string{"hash": from},
	}
	var res struct {
		Name   string `json:"name"`
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	}
	if err := s.c.http.doJSON(ctx, "POST", s.path(slug), nil, body, &res); err != nil {
		return nil, err
	}
	return &Branch{Name: res.Name, Target: res.Target.Hash}, nil
}

func (s *cloudBranches) Delete(ctx context.Context, slug, name string) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", s.path(slug)+"/"+name, nil, nil, "application/json")
	return err
}

func (s *cloudBranches) SetDefault(ctx context.Context, slug, name string) error {
	return fmt.Errorf("set-default is Data Center only")
}

// ---------- pipelines ----------

type cloudPipelines struct{ c *cloudClient }

func (s *cloudPipelines) base(slug string) string {
	return fmt.Sprintf("/2.0/repositories/%s/%s/pipelines", s.c.workspace(), slug)
}

type cloudPipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	State       struct {
		Name   string `json:"name"`
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	Target struct {
		RefName string `json:"ref_name"`
		Commit  struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"target"`
	Creator struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	Links struct {
		HTML struct{ Href string } `json:"html"`
	} `json:"links"`
	CreatedOn time.Time `json:"created_on"`
}

func (p *cloudPipeline) toPipeline() Pipeline {
	return Pipeline{
		UUID:      p.UUID,
		Number:    p.BuildNumber,
		State:     p.State.Name,
		Result:    p.State.Result.Name,
		Ref:       p.Target.RefName,
		Commit:    p.Target.Commit.Hash,
		Creator:   p.Creator.DisplayName,
		WebURL:    p.Links.HTML.Href,
		CreatedAt: p.CreatedOn,
	}
}

func (s *cloudPipelines) List(ctx context.Context, slug string, opts ListOptions) ([]Pipeline, error) {
	params := map[string]string{}
	if opts.Limit > 0 {
		params["pagelen"] = strconv.Itoa(opts.Limit)
	}
	var page cloudPage[cloudPipeline]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Pipeline, len(page.Values))
	for i, p := range page.Values {
		out[i] = p.toPipeline()
	}
	return out, nil
}

func (s *cloudPipelines) Get(ctx context.Context, slug, uuid string) (*Pipeline, error) {
	var p cloudPipeline
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug)+"/"+uuid, nil, nil, &p); err != nil {
		return nil, err
	}
	out := p.toPipeline()
	return &out, nil
}

func (s *cloudPipelines) Run(ctx context.Context, slug, ref string, vars map[string]string) (*Pipeline, error) {
	body := map[string]any{
		"target": map[string]any{
			"ref_type": "branch",
			"type":     "pipeline_ref_target",
			"ref_name": ref,
		},
	}
	if len(vars) > 0 {
		varList := make([]map[string]any, 0, len(vars))
		for k, v := range vars {
			varList = append(varList, map[string]any{"key": k, "value": v})
		}
		body["variables"] = varList
	}
	var p cloudPipeline
	if err := s.c.http.doJSON(ctx, "POST", s.base(slug), nil, body, &p); err != nil {
		return nil, err
	}
	out := p.toPipeline()
	return &out, nil
}

func (s *cloudPipelines) Cancel(ctx context.Context, slug, uuid string) error {
	_, err := s.c.http.doRaw(ctx, "POST", s.base(slug)+"/"+uuid+"/stopPipeline", nil, map[string]any{}, "application/json")
	return err
}

func (s *cloudPipelines) Logs(ctx context.Context, slug, uuid string) (io.ReadCloser, error) {
	// Caller must pick a step; we return the first step's logs here.
	var stepsPage cloudPage[struct {
		UUID string `json:"uuid"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug)+"/"+uuid+"/steps", nil, nil, &stepsPage); err != nil {
		return nil, err
	}
	if len(stepsPage.Values) == 0 {
		return nil, fmt.Errorf("pipeline %s has no steps yet", uuid)
	}
	return s.c.http.doStream(ctx, "GET", s.base(slug)+"/"+uuid+"/steps/"+stepsPage.Values[0].UUID+"/log", nil, nil, "text/plain")
}

// ---------- issues ----------

type cloudIssues struct{ c *cloudClient }

func (s *cloudIssues) base(slug string) string {
	return fmt.Sprintf("/2.0/repositories/%s/%s/issues", s.c.workspace(), slug)
}

type cloudIssue struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	State    string `json:"state"`
	Kind     string `json:"kind"`
	Priority string `json:"priority"`
	Reporter struct {
		DisplayName string `json:"display_name"`
	} `json:"reporter"`
	Assignee struct {
		DisplayName string `json:"display_name"`
	} `json:"assignee"`
	Links struct {
		HTML struct{ Href string } `json:"html"`
	} `json:"links"`
	CreatedOn time.Time `json:"created_on"`
}

func (i *cloudIssue) toIssue() Issue {
	return Issue{
		ID:        i.ID,
		Title:     i.Title,
		State:     i.State,
		Kind:      i.Kind,
		Priority:  i.Priority,
		Reporter:  i.Reporter.DisplayName,
		Assignee:  i.Assignee.DisplayName,
		WebURL:    i.Links.HTML.Href,
		CreatedAt: i.CreatedOn,
	}
}

func (s *cloudIssues) List(ctx context.Context, slug string, opts ListOptions) ([]Issue, error) {
	params := map[string]string{}
	if opts.Limit > 0 {
		params["pagelen"] = strconv.Itoa(opts.Limit)
	}
	if opts.Query != "" {
		params["q"] = opts.Query
	}
	var page cloudPage[cloudIssue]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug), params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Issue, len(page.Values))
	for i, v := range page.Values {
		out[i] = v.toIssue()
	}
	return out, nil
}

func (s *cloudIssues) Get(ctx context.Context, slug string, id int) (*Issue, error) {
	var v cloudIssue
	if err := s.c.http.doJSON(ctx, "GET", fmt.Sprintf("%s/%d", s.base(slug), id), nil, nil, &v); err != nil {
		return nil, err
	}
	out := v.toIssue()
	return &out, nil
}

func (s *cloudIssues) Create(ctx context.Context, slug, title, body, kind, priority string) (*Issue, error) {
	payload := map[string]any{
		"title":    title,
		"content":  map[string]string{"raw": body},
		"kind":     kind,
		"priority": priority,
	}
	var v cloudIssue
	if err := s.c.http.doJSON(ctx, "POST", s.base(slug), nil, payload, &v); err != nil {
		return nil, err
	}
	out := v.toIssue()
	return &out, nil
}

func (s *cloudIssues) Close(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "PUT", fmt.Sprintf("%s/%d", s.base(slug), id), nil, map[string]any{"state": "closed"}, "application/json")
	return err
}

func (s *cloudIssues) Reopen(ctx context.Context, slug string, id int) error {
	_, err := s.c.http.doRaw(ctx, "PUT", fmt.Sprintf("%s/%d", s.base(slug), id), nil, map[string]any{"state": "new"}, "application/json")
	return err
}

func (s *cloudIssues) Comment(ctx context.Context, slug string, id int, body string) (*Comment, error) {
	payload := map[string]any{"content": map[string]string{"raw": body}}
	var res struct {
		ID        int       `json:"id"`
		CreatedOn time.Time `json:"created_on"`
		User      struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
	}
	if err := s.c.http.doJSON(ctx, "POST", fmt.Sprintf("%s/%d/comments", s.base(slug), id), nil, payload, &res); err != nil {
		return nil, err
	}
	return &Comment{ID: res.ID, Author: res.User.DisplayName, Body: res.Content.Raw, CreatedAt: res.CreatedOn}, nil
}

// ---------- webhooks ----------

type cloudWebhooks struct{ c *cloudClient }

func (s *cloudWebhooks) base(slug string) string {
	return fmt.Sprintf("/2.0/repositories/%s/%s/hooks", s.c.workspace(), slug)
}

func (s *cloudWebhooks) List(ctx context.Context, slug string) ([]Webhook, error) {
	var page cloudPage[struct {
		UUID        string   `json:"uuid"`
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Events      []string `json:"events"`
		Active      bool     `json:"active"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug), nil, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Webhook, 0, len(page.Values))
	for _, w := range page.Values {
		out = append(out, Webhook{ID: w.UUID, URL: w.URL, Events: w.Events, Name: w.Description, Active: w.Active})
	}
	return out, nil
}

func (s *cloudWebhooks) Create(ctx context.Context, slug, name, url string, events []string) (*Webhook, error) {
	payload := map[string]any{
		"description": name,
		"url":         url,
		"active":      true,
		"events":      events,
	}
	var w struct {
		UUID        string   `json:"uuid"`
		URL         string   `json:"url"`
		Description string   `json:"description"`
		Events      []string `json:"events"`
		Active      bool     `json:"active"`
	}
	if err := s.c.http.doJSON(ctx, "POST", s.base(slug), nil, payload, &w); err != nil {
		return nil, err
	}
	return &Webhook{ID: w.UUID, URL: w.URL, Events: w.Events, Name: w.Description, Active: w.Active}, nil
}

func (s *cloudWebhooks) Delete(ctx context.Context, slug, id string) error {
	_, err := s.c.http.doRaw(ctx, "DELETE", s.base(slug)+"/"+id, nil, nil, "application/json")
	return err
}

// ---------- compare ----------

type cloudCompare struct{ c *cloudClient }

func (s *cloudCompare) base(slug string) string {
	return fmt.Sprintf("/2.0/repositories/%s/%s", s.c.workspace(), slug)
}

// spec returns "from..to" or "from...to" per three-dot semantics.
func (s *cloudCompare) spec(from, to string, threeDot bool) string {
	sep := ".."
	if threeDot {
		sep = "..."
	}
	// Bitbucket Cloud expects target..source (newer-first). We reorder
	// so the caller can pass `from` (base) and `to` (head) naturally.
	return to + sep + from
}

func (s *cloudCompare) Diff(ctx context.Context, slug, from, to string, threeDot bool) (io.ReadCloser, error) {
	return s.c.http.doStream(ctx, "GET", s.base(slug)+"/diff/"+s.spec(from, to, threeDot), nil, nil, "text/plain")
}

func (s *cloudCompare) Commits(ctx context.Context, slug, from, to string) ([]Commit, error) {
	// /commits/{to}?exclude={from} yields the ahead-only commits.
	params := map[string]string{"exclude": from}
	var page cloudPage[struct {
		Hash    string    `json:"hash"`
		Message string    `json:"message"`
		Date    time.Time `json:"date"`
		Author  struct {
			Raw  string `json:"raw"`
			User struct {
				DisplayName string `json:"display_name"`
			} `json:"user"`
		} `json:"author"`
	}]
	if err := s.c.http.doJSON(ctx, "GET", s.base(slug)+"/commits/"+to, params, nil, &page); err != nil {
		return nil, err
	}
	out := make([]Commit, 0, len(page.Values))
	for _, v := range page.Values {
		name := v.Author.User.DisplayName
		if name == "" {
			name = v.Author.Raw
		}
		short := v.Hash
		if len(short) > 7 {
			short = short[:7]
		}
		out = append(out, Commit{
			SHA: v.Hash, ShortSHA: short,
			Author: name, Message: v.Message, CreatedAt: v.Date,
		})
	}
	return out, nil
}

// ---------- raw ----------

type rawCloud struct{ c *cloudClient }

func (r *rawCloud) Do(ctx context.Context, method, path string, params map[string]string, body io.Reader) ([]byte, error) {
	return r.c.http.doRaw(ctx, method, path, params, body, "application/json")
}
