package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

func newRepoCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "repo",
		Aliases: []string{"repository"},
		Short:   "Manage Bitbucket repositories",
	}
	cmd.AddCommand(
		newRepoListCmd(g),
		newRepoViewCmd(g),
		newRepoCreateCmd(g),
		newRepoCloneCmd(g),
		newRepoBrowseCmd(g),
		newRepoDeleteCmd(g),
	)
	return cmd
}

func newRepoListCmd(g *GlobalFlags) *cobra.Command {
	var limit int
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repositories in the active workspace/project",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			repos, err := cl.Repos().List(withCtx(), bitbucket.ListOptions{Limit: limit, Query: query})
			if err != nil {
				return err
			}
			cols := []string{"SLUG", "NAME", "DEFAULT", "UPDATED"}
			rows := make([][]string, 0, len(repos))
			for _, r := range repos {
				rows = append(rows, []string{r.Slug, r.Name, r.DefaultBranch, r.UpdatedAt.Format("2006-01-02")})
			}
			return renderWith(g, repos, cols, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().StringVar(&query, "query", "", "server-side filter")
	return cmd
}

func newRepoViewCmd(g *GlobalFlags) *cobra.Command {
	var web bool
	cmd := &cobra.Command{
		Use:   "view [slug]",
		Short: "View repository details",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug(firstArg(args))
			if err != nil {
				return err
			}
			r, err := cl.Repos().Get(withCtx(), slug)
			if err != nil {
				return err
			}
			if web {
				return browser.OpenURL(r.WebURL)
			}
			return renderValueWith(g, r)
		},
	}
	cmd.Flags().BoolVar(&web, "web", false, "open in browser")
	return cmd
}

func newRepoCreateCmd(g *GlobalFlags) *cobra.Command {
	var description string
	var public bool
	cmd := &cobra.Command{
		Use:   "create <slug>",
		Short: "Create a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			r, err := cl.Repos().Create(withCtx(), args[0], description, !public)
			if err != nil {
				return err
			}
			stderrf("Created %s", r.WebURL)
			return renderValueWith(g, r)
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "repo description")
	cmd.Flags().BoolVar(&public, "public", false, "create a public repo")
	return cmd
}

func newRepoCloneCmd(g *GlobalFlags) *cobra.Command {
	var useSSH bool
	var dir string
	cmd := &cobra.Command{
		Use:   "clone <slug> [dir]",
		Short: "Clone a repository via git",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			r, err := cl.Repos().Get(withCtx(), args[0])
			if err != nil {
				return err
			}
			url := r.HTTPSCloneURL
			if useSSH && r.SSHCloneURL != "" {
				url = r.SSHCloneURL
			}
			if url == "" {
				return fmt.Errorf("no clone URL advertised for %s", r.Slug)
			}
			if len(args) == 2 {
				dir = args[1]
			}
			cmdArgs := []string{"clone", url}
			if dir != "" {
				cmdArgs = append(cmdArgs, dir)
			}
			c := exec.Command("git", cmdArgs...)
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			c.Stdin = os.Stdin
			return c.Run()
		},
	}
	cmd.Flags().BoolVar(&useSSH, "ssh", false, "clone via SSH instead of HTTPS")
	return cmd
}

func newRepoBrowseCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "browse [slug]",
		Short: "Open the repository in a browser",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug(firstArg(args))
			if err != nil {
				return err
			}
			r, err := cl.Repos().Get(withCtx(), slug)
			if err != nil {
				return err
			}
			return browser.OpenURL(r.WebURL)
		},
	}
}

func newRepoDeleteCmd(g *GlobalFlags) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a repository (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if prompt(fmt.Sprintf("Really delete %s? [y/N] ", args[0])) != "y" {
					return fmt.Errorf("aborted")
				}
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			return cl.Repos().Delete(withCtx(), args[0])
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip confirmation")
	return cmd
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

// renderWith / renderValueWith are thin shims around output.New so every
// verb above stays readable.
func renderWith(g *GlobalFlags, value any, cols []string, rows [][]string) error {
	return output_Render(g, value, cols, rows)
}
func renderValueWith(g *GlobalFlags, value any) error {
	return output_RenderValue(g, value)
}
