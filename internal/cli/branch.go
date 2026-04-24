package cli

import (
	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
	"github.com/spf13/cobra"
)

func newBranchCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Manage repository branches",
	}
	cmd.AddCommand(
		newBranchListCmd(g),
		newBranchCreateCmd(g),
		newBranchDeleteCmd(g),
		newBranchSetDefaultCmd(g),
	)
	return cmd
}

func newBranchListCmd(g *GlobalFlags) *cobra.Command {
	var limit int
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			bs, err := cl.Branches().List(withCtx(), slug, bitbucket.ListOptions{Limit: limit, Query: query})
			if err != nil {
				return err
			}
			cols := []string{"NAME", "DEFAULT", "TARGET"}
			rows := make([][]string, 0, len(bs))
			for _, b := range bs {
				def := ""
				if b.IsDefault {
					def = "*"
				}
				rows = append(rows, []string{b.Name, def, b.Target})
			}
			return renderWith(g, bs, cols, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "page size")
	cmd.Flags().StringVar(&query, "query", "", "filter")
	return cmd
}

func newBranchCreateCmd(g *GlobalFlags) *cobra.Command {
	var from string
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			if from == "" {
				from = "main"
			}
			b, err := cl.Branches().Create(withCtx(), slug, args[0], from)
			if err != nil {
				return err
			}
			return renderValueWith(g, b)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "start point (branch or SHA; default: main)")
	return cmd
}

func newBranchDeleteCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a branch",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.Branches().Delete(withCtx(), slug, args[0])
		},
	}
}

func newBranchSetDefaultCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "set-default <name>",
		Short: "Set the default branch (Data Center)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.Branches().SetDefault(withCtx(), slug, args[0])
		},
	}
}
