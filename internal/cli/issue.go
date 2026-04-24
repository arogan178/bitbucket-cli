package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
)

func newIssueCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issue",
		Short: "Manage Bitbucket Cloud issues",
	}
	cmd.AddCommand(
		newIssueListCmd(g),
		newIssueViewCmd(g),
		newIssueCreateCmd(g),
		newIssueCloseCmd(g),
		newIssueReopenCmd(g),
		newIssueCommentCmd(g),
	)
	return cmd
}

func newIssueListCmd(g *GlobalFlags) *cobra.Command {
	var limit int
	var query string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			issues, err := cl.Issues().List(withCtx(), slug, bitbucket.ListOptions{Limit: limit, Query: query})
			if err != nil {
				return err
			}
			cols := []string{"ID", "TITLE", "KIND", "STATE", "ASSIGNEE"}
			rows := make([][]string, 0, len(issues))
			for _, i := range issues {
				rows = append(rows, []string{strconv.Itoa(i.ID), truncate(i.Title, 60), i.Kind, i.State, i.Assignee})
			}
			return renderWith(g, issues, cols, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 30, "page size")
	cmd.Flags().StringVar(&query, "query", "", "filter")
	return cmd
}

func newIssueViewCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "view <id>",
		Short: "View an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue id %q", args[0])
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			i, err := cl.Issues().Get(withCtx(), slug, id)
			if err != nil {
				return err
			}
			return renderValueWith(g, i)
		},
	}
}

func newIssueCreateCmd(g *GlobalFlags) *cobra.Command {
	var title, body, kind, priority string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an issue",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("--title is required")
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			i, err := cl.Issues().Create(withCtx(), slug, title, body, kind, priority)
			if err != nil {
				return err
			}
			return renderValueWith(g, i)
		},
	}
	cmd.Flags().StringVarP(&title, "title", "t", "", "issue title (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "issue body")
	cmd.Flags().StringVarP(&kind, "kind", "k", "bug", "bug | enhancement | proposal | task")
	cmd.Flags().StringVarP(&priority, "priority", "p", "major", "trivial | minor | major | critical | blocker")
	return cmd
}

func newIssueCloseCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "close <id>",
		Short: "Close an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue id %q", args[0])
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.Issues().Close(withCtx(), slug, id)
		},
	}
}

func newIssueReopenCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "reopen <id>",
		Short: "Reopen an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue id %q", args[0])
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			return cl.Issues().Reopen(withCtx(), slug, id)
		},
	}
}

func newIssueCommentCmd(g *GlobalFlags) *cobra.Command {
	var body string
	cmd := &cobra.Command{
		Use:   "comment <id>",
		Short: "Comment on an issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue id %q", args[0])
			}
			if body == "" {
				return fmt.Errorf("--body is required")
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			_, err = cl.Issues().Comment(withCtx(), slug, id, body)
			return err
		},
	}
	cmd.Flags().StringVarP(&body, "body", "b", "", "comment body (required)")
	return cmd
}
