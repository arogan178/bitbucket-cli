// Package bitbucket defines a backend-agnostic client for Bitbucket Cloud
// (api.bitbucket.org/2.0) and Bitbucket Data Center (rest/api/1.0). The
// CLI and TUI depend only on the Client interface.
package bitbucket

import "time"

// Repo is the normalised repository view.
type Repo struct {
	Slug          string    `json:"slug"`
	Name          string    `json:"name"`
	Project       string    `json:"project,omitempty"`
	Workspace     string    `json:"workspace,omitempty"`
	Description   string    `json:"description,omitempty"`
	DefaultBranch string    `json:"default_branch,omitempty"`
	HTTPSCloneURL string    `json:"https_clone_url,omitempty"`
	SSHCloneURL   string    `json:"ssh_clone_url,omitempty"`
	WebURL        string    `json:"web_url,omitempty"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
	Private       bool      `json:"private"`
}

// PullRequest is the normalised PR view.
type PullRequest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	State       string    `json:"state"` // OPEN, MERGED, DECLINED, SUPERSEDED
	Author      string    `json:"author,omitempty"`
	Source      string    `json:"source_branch"`
	Target      string    `json:"target_branch"`
	WebURL      string    `json:"web_url,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	Reviewers   []string  `json:"reviewers,omitempty"`
	Approvals   []string  `json:"approvals,omitempty"`
	Draft       bool      `json:"draft,omitempty"`
}

// Branch is the normalised branch view.
type Branch struct {
	Name      string `json:"name"`
	Target    string `json:"target,omitempty"` // commit sha
	IsDefault bool   `json:"default,omitempty"`
}

// Pipeline is the normalised pipeline/build view.
type Pipeline struct {
	UUID      string    `json:"uuid"`
	Number    int       `json:"number,omitempty"`
	State     string    `json:"state"`            // RUNNING, COMPLETED, PAUSED, etc.
	Result    string    `json:"result,omitempty"` // SUCCESSFUL, FAILED, STOPPED
	Ref       string    `json:"ref,omitempty"`
	Commit    string    `json:"commit,omitempty"`
	Creator   string    `json:"creator,omitempty"`
	WebURL    string    `json:"web_url,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Comment is a PR or issue comment.
type Comment struct {
	ID        int       `json:"id"`
	Author    string    `json:"author,omitempty"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Issue is the normalised issue view (Cloud only).
type Issue struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Kind      string    `json:"kind,omitempty"`
	Priority  string    `json:"priority,omitempty"`
	Reporter  string    `json:"reporter,omitempty"`
	Assignee  string    `json:"assignee,omitempty"`
	WebURL    string    `json:"web_url,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Webhook is a repo-level webhook.
type Webhook struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
	Name   string   `json:"name,omitempty"`
	Active bool     `json:"active"`
}

// Commit is a normalised git commit summary.
type Commit struct {
	SHA       string    `json:"sha"`
	ShortSHA  string    `json:"short_sha,omitempty"`
	Author    string    `json:"author,omitempty"`
	Email     string    `json:"email,omitempty"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// BuildStatus is one commit build result (PR checks).
type BuildStatus struct {
	Key         string `json:"key"`
	Name        string `json:"name,omitempty"`
	State       string `json:"state"` // SUCCESSFUL, FAILED, INPROGRESS, STOPPED
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
}

// ListOptions is the pagination envelope used by list calls.
type ListOptions struct {
	Limit  int
	Page   int
	Query  string // server-side filter (role=member, q=..., filter=...)
	Cursor string // some Cloud endpoints use opaque cursors
}

// CreatePullRequestInput is the input to PullRequests.Create.
type CreatePullRequestInput struct {
	Title       string
	Description string
	Source      string
	Target      string
	Reviewers   []string
	CloseSource bool
	Draft       bool
}

// EditPullRequestInput is the input to PullRequests.Edit.
type EditPullRequestInput struct {
	Title       *string
	Description *string
	Target      *string
}

// MergeOptions controls merge strategy.
type MergeOptions struct {
	Strategy    string // merge-commit | squash | fast-forward
	Message     string
	CloseSource bool
}
