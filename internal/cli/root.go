// Package cli wires all `bt` subcommands together. The root command owns
// global flags (--context, --repo, --workspace, --project, --json, --yaml,
// --template, --jq) plus shared helpers that resolve the active context,
// credentials, and client.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
	"github.com/arogan178/bitbucket-cli/internal/config"
	"github.com/arogan178/bitbucket-cli/internal/output"
)

// BuildInfo is populated at build time.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// GlobalFlags holds values bound at the root level.
type GlobalFlags struct {
	ContextName string
	Repo        string
	Workspace   string
	Project     string
	JSON        bool
	YAML        bool
	Template    string
	JQ          string
}

// NewRootCmd builds the cobra command tree.
func NewRootCmd(info BuildInfo) *cobra.Command {
	g := &GlobalFlags{}

	root := &cobra.Command{
		Use:   "bt",
		Short: "bt — a gh-style CLI for Bitbucket",
		Long: `bt is a gh-style single-binary CLI for Bitbucket Cloud and Data Center.

Use subcommands (auth, context, repo, pr, branch, compare, issue, webhook,
pipeline, api) to script Bitbucket from your shell. Structured output is
available via --json, --yaml, --template, and --jq on every command.`,
		Version: fmt.Sprintf("%s (commit %s, built %s)", info.Version, info.Commit, info.Date),
		// Cobra would otherwise print `Error: ...` itself, and main
		// also prints `error: ...` + sets exit 1. Silence cobra's copy
		// so the user sees exactly one line.
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&g.ContextName, "context", "", "override the active context")
	root.PersistentFlags().StringVar(&g.Repo, "repo", "", "override the repo slug")
	root.PersistentFlags().StringVar(&g.Workspace, "workspace", "", "override the Bitbucket Cloud workspace")
	root.PersistentFlags().StringVar(&g.Project, "project", "", "override the Bitbucket DC project key")
	root.PersistentFlags().BoolVar(&g.JSON, "json", false, "output as JSON")
	root.PersistentFlags().BoolVar(&g.YAML, "yaml", false, "output as YAML")
	root.PersistentFlags().StringVar(&g.Template, "template", "", "render with a Go text/template")
	root.PersistentFlags().StringVar(&g.JQ, "jq", "", "filter output through a jq expression")

	root.AddCommand(
		newAuthCmd(g),
		newContextCmd(g),
		newRepoCmd(g),
		newPRCmd(g),
		newBranchCmd(g),
		newCompareCmd(g),
		newIssueCmd(g),
		newWebhookCmd(g),
		newPipelineCmd(g),
		newAPICmd(g),
	)

	return root
}

// outputOpts converts CLI global flags into output.Options.
func (g *GlobalFlags) outputOpts() output.Options {
	return output.Options{
		JSON:     g.JSON,
		YAML:     g.YAML,
		Template: g.Template,
		JQ:       g.JQ,
	}
}

// resolveContext returns the active context, applying flag overrides.
func (g *GlobalFlags) resolveContext() (*config.Context, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	ctx, err := cfg.ActiveContext(g.ContextName)
	if err != nil {
		return nil, err
	}
	// Apply one-shot overrides so commands see the final values without
	// persisting them.
	c := *ctx
	if g.Workspace != "" {
		c.Workspace = g.Workspace
	}
	if g.Project != "" {
		c.Project = g.Project
	}
	return &c, nil
}

// client resolves context + credential and returns a backend client.
func (g *GlobalFlags) client() (bitbucket.Client, *config.Context, error) {
	ctx, err := g.resolveContext()
	if err != nil {
		return nil, nil, err
	}
	cred, err := auth.Load(ctx)
	if err != nil {
		return nil, nil, err
	}
	cl, err := bitbucket.New(ctx, cred)
	if err != nil {
		return nil, nil, err
	}
	return cl, ctx, nil
}

// repoSlug returns the repo slug to use (flag > positional > context default).
func (g *GlobalFlags) repoSlug(positional string) (string, error) {
	if positional != "" {
		return positional, nil
	}
	if g.Repo != "" {
		return g.Repo, nil
	}
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}
	ctx, err := cfg.ActiveContext(g.ContextName)
	if err != nil {
		return "", err
	}
	if ctx.Repo != "" {
		return ctx.Repo, nil
	}
	return "", fmt.Errorf("no repo specified; use --repo <slug>, pass a positional, or set context default")
}

// withCtx creates a fresh context for API calls. In the future we may
// thread timeouts / signal handling here.
func withCtx() context.Context { return context.Background() }

// stderrf writes a formatted status line to stderr.
func stderrf(format string, args ...any) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
}
