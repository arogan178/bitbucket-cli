package bitbucket

import (
	"context"
	"io"

	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/config"
)

// Client is the backend-agnostic Bitbucket client. Cloud (v2) and DC (v1)
// both implement it.
type Client interface {
	Kind() config.Kind
	Host() string

	Repos() RepoService
	PullRequests() PullRequestService
	Branches() BranchService
	Compare() CompareService
	Pipelines() PipelineService
	Issues() IssueService
	Webhooks() WebhookService
	Raw() RawService
}

// CompareService produces branch/ref comparisons (diff + commit list)
// without needing an open pull request.
type CompareService interface {
	// Diff returns the unified diff between `from` and `to`. When
	// `threeDot` is true, we use the merge-base (git `from...to`
	// semantics); otherwise the raw diff from `from` to `to`.
	Diff(ctx context.Context, repoSlug, from, to string, threeDot bool) (io.ReadCloser, error)
	// Commits returns the list of commits unique to `to` relative to
	// `from` (three-dot semantics).
	Commits(ctx context.Context, repoSlug, from, to string) ([]Commit, error)
}

// RepoService manages repositories.
type RepoService interface {
	List(ctx context.Context, opts ListOptions) ([]Repo, error)
	Get(ctx context.Context, slug string) (*Repo, error)
	Create(ctx context.Context, slug, description string, private bool) (*Repo, error)
	Delete(ctx context.Context, slug string) error
}

// PullRequestService manages pull requests.
type PullRequestService interface {
	List(ctx context.Context, repoSlug string, opts ListOptions) ([]PullRequest, error)
	Get(ctx context.Context, repoSlug string, id int) (*PullRequest, error)
	Create(ctx context.Context, repoSlug string, in CreatePullRequestInput) (*PullRequest, error)
	Edit(ctx context.Context, repoSlug string, id int, in EditPullRequestInput) (*PullRequest, error)
	Merge(ctx context.Context, repoSlug string, id int, opts MergeOptions) error
	Decline(ctx context.Context, repoSlug string, id int) error
	Approve(ctx context.Context, repoSlug string, id int) error
	Unapprove(ctx context.Context, repoSlug string, id int) error
	Comment(ctx context.Context, repoSlug string, id int, body string) (*Comment, error)
	Comments(ctx context.Context, repoSlug string, id int) ([]Comment, error)
	Checks(ctx context.Context, repoSlug string, id int) ([]BuildStatus, error)
	Diff(ctx context.Context, repoSlug string, id int) (io.ReadCloser, error)
}

// BranchService manages repository branches.
type BranchService interface {
	List(ctx context.Context, repoSlug string, opts ListOptions) ([]Branch, error)
	Create(ctx context.Context, repoSlug, name, from string) (*Branch, error)
	Delete(ctx context.Context, repoSlug, name string) error
	SetDefault(ctx context.Context, repoSlug, name string) error // DC only
}

// PipelineService manages Bitbucket Pipelines (Cloud only).
type PipelineService interface {
	List(ctx context.Context, repoSlug string, opts ListOptions) ([]Pipeline, error)
	Get(ctx context.Context, repoSlug, uuid string) (*Pipeline, error)
	Run(ctx context.Context, repoSlug, ref string, vars map[string]string) (*Pipeline, error)
	Cancel(ctx context.Context, repoSlug, uuid string) error
	Logs(ctx context.Context, repoSlug, uuid string) (io.ReadCloser, error)
}

// IssueService manages Bitbucket Cloud issues.
type IssueService interface {
	List(ctx context.Context, repoSlug string, opts ListOptions) ([]Issue, error)
	Get(ctx context.Context, repoSlug string, id int) (*Issue, error)
	Create(ctx context.Context, repoSlug, title, body, kind, priority string) (*Issue, error)
	Close(ctx context.Context, repoSlug string, id int) error
	Reopen(ctx context.Context, repoSlug string, id int) error
	Comment(ctx context.Context, repoSlug string, id int, body string) (*Comment, error)
}

// WebhookService manages repo-level webhooks.
type WebhookService interface {
	List(ctx context.Context, repoSlug string) ([]Webhook, error)
	Create(ctx context.Context, repoSlug, name, url string, events []string) (*Webhook, error)
	Delete(ctx context.Context, repoSlug, id string) error
}

// RawService exposes a raw API call for endpoints we don't yet wrap.
type RawService interface {
	Do(ctx context.Context, method, path string, params map[string]string, body io.Reader) ([]byte, error)
}

// New constructs a backend-specific client from a context + credential.
func New(ctx *config.Context, cred auth.Credential) (Client, error) {
	switch ctx.Kind {
	case config.KindDataCenter:
		return newDataCenterClient(ctx, cred)
	case config.KindCloud:
		fallthrough
	default:
		return newCloudClient(ctx, cred)
	}
}
