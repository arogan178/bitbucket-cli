package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
	"github.com/arogan178/bitbucket-cli/internal/output"
)

func newPRCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Work with pull requests",
	}
	cmd.AddCommand(
		newPRListCmd(g),
		newPRViewCmd(g),
		newPRCreateCmd(g),
		newPREditCmd(g),
		newPRMergeCmd(g),
		newPRDeclineCmd(g),
		newPRApproveCmd(g),
		newPRUnapproveCmd(g),
		newPRCommentCmd(g),
		newPRChecksCmd(g),
		newPRCheckoutCmd(g),
		newPRDiffCmd(g),
	)
	return cmd
}

func prID(arg string) (int, error) {
	n, err := strconv.Atoi(arg)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid PR id %q", arg)
	}
	return n, nil
}

func newPRListCmd(g *GlobalFlags) *cobra.Command {
	var limit int
	var state string
	var mine bool
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			q := query
			if state != "" {
				if q != "" {
					q += " AND "
				}
				q += fmt.Sprintf(`state="%s"`, state)
			}
			if mine {
				// Cloud-specific; the backend may ignore this for DC.
				if q != "" {
					q += " AND "
				}
				q += "author.uuid=\"{me}\""
			}
			prs, err := cl.PullRequests().List(withCtx(), slug, bitbucket.ListOptions{Limit: limit, Query: q})
			if err != nil {
				return err
			}
			cols := []string{"ID", "TITLE", "SOURCE → TARGET", "STATE", "AUTHOR", "UPDATED"}
			rows := make([][]string, 0, len(prs))
			for _, pr := range prs {
				rows = append(rows, []string{
					strconv.Itoa(pr.ID),
					truncate(pr.Title, 60),
					pr.Source + " → " + pr.Target,
					pr.State,
					pr.Author,
					pr.UpdatedAt.Format("2006-01-02"),
				})
			}
			return renderWith(g, prs, cols, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "page size")
	cmd.Flags().StringVar(&state, "state", "OPEN", "OPEN | MERGED | DECLINED | SUPERSEDED (or empty for all)")
	cmd.Flags().StringVar(&query, "query", "", "raw Bitbucket q= filter")
	cmd.Flags().BoolVar(&mine, "mine", false, "show only PRs I authored")
	return cmd
}

func newPRViewCmd(g *GlobalFlags) *cobra.Command {
	var web bool
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			pr, err := cl.PullRequests().Get(withCtx(), slug, id)
			if err != nil {
				return err
			}
			if web {
				return browser.OpenURL(pr.WebURL)
			}
			return renderValueWith(g, pr)
		},
	}
	cmd.Flags().BoolVar(&web, "web", false, "open in browser")
	return cmd
}

func newPRCreateCmd(g *GlobalFlags) *cobra.Command {
	var title, body, source, target string
	var reviewers []string
	var closeSource, draft bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			if source == "" {
				source = currentGitBranch()
			}
			if source == "" {
				return fmt.Errorf("--source is required (could not detect current branch)")
			}
			if target == "" {
				target = "main"
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			pr, err := cl.PullRequests().Create(withCtx(), slug, bitbucket.CreatePullRequestInput{
				Title: title, Description: body, Source: source, Target: target,
				Reviewers: reviewers, CloseSource: closeSource, Draft: draft,
			})
			if err != nil {
				return err
			}
			stderrf("Created PR #%d — %s", pr.ID, pr.WebURL)
			return renderValueWith(g, pr)
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "description (markdown)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "source branch (default: current branch)")
	cmd.Flags().StringVarP(&target, "target", "T", "", "target branch (default: main)")
	cmd.Flags().StringSliceVarP(&reviewers, "reviewer", "r", nil, "add reviewer (repeatable)")
	cmd.Flags().BoolVar(&closeSource, "delete-branch", false, "delete source branch on merge")
	cmd.Flags().BoolVar(&draft, "draft", false, "create as draft (Cloud only)")
	return cmd
}

func newPREditCmd(g *GlobalFlags) *cobra.Command {
	var title, body, target string
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a pull request title, body, or target branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			in := bitbucket.EditPullRequestInput{}
			if cmd.Flags().Changed("title") {
				in.Title = &title
			}
			if cmd.Flags().Changed("body") {
				in.Description = &body
			}
			if cmd.Flags().Changed("target") {
				in.Target = &target
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			pr, err := cl.PullRequests().Edit(withCtx(), slug, id, in)
			if err != nil {
				return err
			}
			return renderValueWith(g, pr)
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "new title")
	cmd.Flags().StringVarP(&body, "body", "b", "", "new description")
	cmd.Flags().StringVarP(&target, "target", "T", "", "new target branch")
	return cmd
}

func newPRMergeCmd(g *GlobalFlags) *cobra.Command {
	var strategy, message string
	var deleteBranch bool
	cmd := &cobra.Command{
		Use:   "merge <id>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.PullRequests().Merge(withCtx(), slug, id, bitbucket.MergeOptions{
				Strategy: strategy, Message: message, CloseSource: deleteBranch,
			})
		},
	}
	cmd.Flags().StringVar(&strategy, "strategy", "", "merge-commit | squash | fast-forward")
	cmd.Flags().StringVarP(&message, "message", "m", "", "merge commit message")
	cmd.Flags().BoolVar(&deleteBranch, "delete-branch", false, "delete source branch after merge")
	return cmd
}

func newPRDeclineCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "decline <id>",
		Short: "Decline (close) a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.PullRequests().Decline(withCtx(), slug, id)
		},
	}
}

func newPRApproveCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.PullRequests().Approve(withCtx(), slug, id)
		},
	}
}

func newPRUnapproveCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "unapprove <id>",
		Short: "Revoke your approval on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.PullRequests().Unapprove(withCtx(), slug, id)
		},
	}
}

func newPRCommentCmd(g *GlobalFlags) *cobra.Command {
	var text string
	cmd := &cobra.Command{
		Use:   "comment <id>",
		Short: "Add a comment to a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			if text == "" {
				return fmt.Errorf("--text is required")
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			_, err = cl.PullRequests().Comment(withCtx(), slug, id, text)
			return err
		},
	}
	cmd.Flags().StringVarP(&text, "text", "t", "", "comment body (required)")
	return cmd
}

func newPRChecksCmd(g *GlobalFlags) *cobra.Command {
	var wait bool
	var timeout time.Duration
	var failFast bool
	cmd := &cobra.Command{
		Use:   "checks <id>",
		Short: "Show build status for the PR commit; optionally wait for completion",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}

			deadline := time.Now().Add(timeout)
			for {
				checks, err := cl.PullRequests().Checks(withCtx(), slug, id)
				if err != nil {
					return err
				}
				if !wait {
					return renderChecks(g, checks)
				}
				allDone := true
				for _, c := range checks {
					if c.State == "INPROGRESS" || c.State == "IN_PROGRESS" {
						allDone = false
					}
					if failFast && (c.State == "FAILED" || c.State == "STOPPED") {
						_ = renderChecks(g, checks)
						return fmt.Errorf("check %q failed", c.Key)
					}
				}
				if allDone {
					return renderChecks(g, checks)
				}
				if timeout > 0 && time.Now().After(deadline) {
					return fmt.Errorf("timed out waiting for checks")
				}
				time.Sleep(10 * time.Second)
			}
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "poll until all checks complete")
	cmd.Flags().DurationVar(&timeout, "timeout", 0, "max time to wait (0 = no limit)")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "exit on first failure")
	return cmd
}

func renderChecks(g *GlobalFlags, checks []bitbucket.BuildStatus) error {
	cols := []string{"KEY", "STATE", "NAME", "URL"}
	rows := make([][]string, 0, len(checks))
	for _, c := range checks {
		rows = append(rows, []string{c.Key, c.State, c.Name, c.URL})
	}
	return renderWith(g, checks, cols, rows)
}

func newPRCheckoutCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "checkout <id>",
		Short: "Fetch the PR source branch locally into pr/<id>",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			pr, err := cl.PullRequests().Get(withCtx(), slug, id)
			if err != nil {
				return err
			}
			branch := fmt.Sprintf("pr/%d", id)
			// `git fetch origin <source>:pr/<id> && git checkout pr/<id>`
			steps := [][]string{
				{"git", "fetch", "origin", pr.Source + ":" + branch},
				{"git", "checkout", branch},
			}
			for _, s := range steps {
				c := exec.Command(s[0], s[1:]...)
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newPRDiffCmd(g *GlobalFlags) *cobra.Command {
	var colorFlag string
	var noPager bool
	var stat bool
	cmd := &cobra.Command{
		Use:   "diff <id>",
		Short: "Show the diff for a pull request, colourised and paged",
		Long: `Show the unified diff for a pull request.

Colour is auto-detected from TTY; override with --color=always|never (gh-style).
The diff is paged through $BT_PAGER, $PAGER, delta, or less -R when writing
to a terminal; use --no-pager to stream straight to stdout. Pipe-friendly:
bt pr diff 42 | patch -p1 works because we don't colour when stdout isn't a
terminal.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := prID(args[0])
			if err != nil {
				return err
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			r, err := cl.PullRequests().Diff(withCtx(), slug, id)
			if err != nil {
				return err
			}
			defer r.Close()
			if stat {
				return output.RenderDiffStat(r, os.Stdout, output.ParseColorMode(colorFlag))
			}
			return writeDiff(r, colorFlag, noPager)
		},
	}
	cmd.Flags().StringVar(&colorFlag, "color", "auto", "colour output: auto | always | never")
	cmd.Flags().BoolVar(&noPager, "no-pager", false, "do not page even when stdout is a TTY")
	cmd.Flags().BoolVar(&stat, "stat", false, "show a diffstat summary instead of the full diff")
	return cmd
}

// writeDiff colours the diff and pages it when stdout is a TTY.
func writeDiff(r io.Reader, colorFlag string, noPager bool) error {
	mode := output.ParseColorMode(colorFlag)
	if noPager {
		return output.RenderDiff(r, os.Stdout, mode)
	}
	w, closePager, err := output.OpenPager()
	if err != nil {
		return err
	}
	defer func() { _ = closePager() }()
	return output.RenderDiff(r, w, mode)
}

// currentGitBranch returns the checked-out branch name, or "" on failure.
func currentGitBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return trim(string(out))
}

func trim(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
