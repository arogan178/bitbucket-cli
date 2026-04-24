package cli

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
)

func newPipelineCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pipeline",
		Aliases: []string{"pipe"},
		Short:   "Manage Bitbucket Cloud pipelines",
	}
	cmd.AddCommand(
		newPipelineListCmd(g),
		newPipelineViewCmd(g),
		newPipelineRunCmd(g),
		newPipelineCancelCmd(g),
		newPipelineLogsCmd(g),
	)
	return cmd
}

func newPipelineListCmd(g *GlobalFlags) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent pipelines",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			ps, err := cl.Pipelines().List(withCtx(), slug, bitbucket.ListOptions{Limit: limit})
			if err != nil {
				return err
			}
			cols := []string{"UUID", "STATE", "RESULT", "REF", "CREATED"}
			rows := make([][]string, 0, len(ps))
			for _, p := range ps {
				rows = append(rows, []string{p.UUID, p.State, p.Result, p.Ref, p.CreatedAt.Format("2006-01-02 15:04")})
			}
			return renderWith(g, ps, cols, rows)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "page size")
	return cmd
}

func newPipelineViewCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "view <uuid>",
		Short: "View a pipeline run",
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
			p, err := cl.Pipelines().Get(withCtx(), slug, args[0])
			if err != nil {
				return err
			}
			return renderValueWith(g, p)
		},
	}
}

func newPipelineRunCmd(g *GlobalFlags) *cobra.Command {
	var ref string
	var vars []string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a pipeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			kv := map[string]string{}
			for _, v := range vars {
				parts := strings.SplitN(v, "=", 2)
				if len(parts) == 2 {
					kv[parts[0]] = parts[1]
				}
			}
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			p, err := cl.Pipelines().Run(withCtx(), slug, ref, kv)
			if err != nil {
				return err
			}
			return renderValueWith(g, p)
		},
	}
	cmd.Flags().StringVar(&ref, "ref", "main", "branch to run against")
	cmd.Flags().StringSliceVar(&vars, "var", nil, "KEY=VALUE variable (repeatable)")
	return cmd
}

func newPipelineCancelCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <uuid>",
		Short: "Stop a running pipeline",
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
			return cl.Pipelines().Cancel(withCtx(), slug, args[0])
		},
	}
}

func newPipelineLogsCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "logs <uuid>",
		Short: "Print logs for the first step of a pipeline",
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
			r, err := cl.Pipelines().Logs(withCtx(), slug, args[0])
			if err != nil {
				return err
			}
			defer r.Close()
			_, err = io.Copy(os.Stdout, r)
			return err
		},
	}
}
