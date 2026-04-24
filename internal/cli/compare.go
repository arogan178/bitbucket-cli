package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/arogan178/bitbucket-cli/internal/output"
)

// newCompareCmd exposes `bt compare <base>..<head>` (two-dot) and
// `<base>...<head>` (three-dot, merge-base). Mirrors `gh` semantics:
// two-dot = "what does head look like relative to base right now", and
// three-dot = "what is new on head since it diverged from base".
//
// This is the main reason the user wanted this CLI: quickly inspect
// diffs between branches without opening a PR.
func newCompareCmd(g *GlobalFlags) *cobra.Command {
	var colorFlag string
	var noPager bool
	var stat bool
	var commits bool
	var repoFlag string

	cmd := &cobra.Command{
		Use:   "compare <base>..<head> | <base>...<head>",
		Short: "Show the diff between two branches, tags, or commits",
		Long: `Compare two refs in a Bitbucket repository.

  bt compare main..feature/foo      two-dot: diff from main directly to feature/foo
  bt compare main...feature/foo     three-dot: what's new on feature/foo since it diverged from main

Without a spec, compares the current branch against its upstream. With
--commits, lists the commits in the range instead of the diff. Pair
--stat with a regular compare call to see a per-file summary.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug := repoFlag
			if slug == "" {
				slug, err = g.repoSlug("")
				if err != nil {
					return err
				}
			}

			spec := ""
			if len(args) == 1 {
				spec = args[0]
			}
			base, head, threeDot, err := parseCompareSpec(spec)
			if err != nil {
				return err
			}

			if commits {
				cs, err := cl.Compare().Commits(withCtx(), slug, base, head)
				if err != nil {
					return err
				}
				cols := []string{"SHA", "AUTHOR", "DATE", "MESSAGE"}
				rows := make([][]string, 0, len(cs))
				for _, c := range cs {
					msg := firstLine(c.Message)
					rows = append(rows, []string{c.ShortSHA, c.Author, c.CreatedAt.Format("2006-01-02"), truncate(msg, 72)})
				}
				return renderWith(g, cs, cols, rows)
			}

			r, err := cl.Compare().Diff(withCtx(), slug, base, head, threeDot)
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
	cmd.Flags().BoolVar(&commits, "commits", false, "list commits in the range instead of the diff")
	cmd.Flags().StringVar(&repoFlag, "repo", "", "override the repo slug")
	return cmd
}

// parseCompareSpec accepts "base..head", "base...head", or an empty
// string. Returns base, head, and whether three-dot semantics were
// requested. When spec is empty, we try to read the current branch and
// its upstream via git.
func parseCompareSpec(spec string) (base, head string, threeDot bool, err error) {
	if spec == "" {
		head = currentGitBranch()
		if head == "" {
			return "", "", false, fmt.Errorf("no compare spec supplied and no current git branch; pass <base>..<head>")
		}
		base = gitUpstream(head)
		if base == "" {
			base = "main"
		}
		// Default to three-dot for "what's new on my branch" feel.
		return base, head, true, nil
	}

	if strings.Contains(spec, "...") {
		parts := strings.SplitN(spec, "...", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", false, fmt.Errorf("invalid spec %q; expected base...head", spec)
		}
		return parts[0], parts[1], true, nil
	}
	if strings.Contains(spec, "..") {
		parts := strings.SplitN(spec, "..", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", false, fmt.Errorf("invalid spec %q; expected base..head", spec)
		}
		return parts[0], parts[1], false, nil
	}
	return "", "", false, fmt.Errorf("spec %q must contain .. or ...", spec)
}

// gitUpstream returns the upstream tracking branch's short name (e.g.
// "origin/main" becomes "main"), empty string on failure.
func gitUpstream(branch string) string {
	// `git rev-parse --abbrev-ref <branch>@{u}` yields the upstream.
	out, err := runGit("rev-parse", "--abbrev-ref", branch+"@{u}")
	if err != nil {
		return ""
	}
	s := trim(out)
	if idx := strings.Index(s, "/"); idx >= 0 {
		return s[idx+1:]
	}
	return s
}

func runGit(args ...string) (string, error) {
	data, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
