package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/config"
	"github.com/arogan178/bitbucket-cli/internal/output"
)

func newContextCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "Manage named Bitbucket contexts (host + kind + defaults)",
	}
	cmd.AddCommand(
		newContextListCmd(g),
		newContextUseCmd(g),
		newContextShowCmd(g),
		newContextCreateCmd(g),
		newContextDeleteCmd(g),
	)
	return cmd
}

func newContextListCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cols := []string{"ACTIVE", "NAME", "HOST", "KIND", "WORKSPACE", "PROJECT"}
			rows := make([][]string, 0, len(cfg.Contexts))
			for _, c := range cfg.Contexts {
				active := ""
				if c.Name == cfg.Active {
					active = "*"
				}
				rows = append(rows, []string{active, c.Name, c.Host, string(c.Kind), c.Workspace, c.Project})
			}
			return output.New(g.outputOpts()).Render(cfg.Contexts, cols, rows)
		},
	}
}

func newContextUseCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Switch the active context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if cfg.Find(args[0]) == nil {
				return fmt.Errorf("context %q does not exist", args[0])
			}
			cfg.Active = args[0]
			if err := cfg.Save(); err != nil {
				return err
			}
			stderrf("Active context is now %q", args[0])
			return nil
		},
	}
}

func newContextShowCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show details of a context (default: active)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := cfg.Active
			if len(args) == 1 {
				name = args[0]
			}
			c := cfg.Find(name)
			if c == nil {
				return fmt.Errorf("context %q not found", name)
			}
			return output.New(g.outputOpts()).RenderValue(c)
		},
	}
}

func newContextCreateCmd(g *GlobalFlags) *cobra.Command {
	var (
		host, kind, workspace, project string
		setActive                       bool
	)
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a context without storing credentials (use `bt auth login` for that)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			k := config.Kind(kind)
			if k == "" {
				k = config.KindCloud
			}
			if host == "" {
				if k == config.KindCloud {
					host = "https://bitbucket.org"
				} else {
					return fmt.Errorf("--host is required for Data Center contexts")
				}
			}
			cfg.Upsert(config.Context{
				Name:      args[0],
				Host:      host,
				Kind:      k,
				Workspace: workspace,
				Project:   project,
			})
			if setActive || cfg.Active == "" {
				cfg.Active = args[0]
			}
			return cfg.Save()
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Bitbucket host")
	cmd.Flags().StringVar(&kind, "kind", "", "cloud | dc (default cloud)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Cloud workspace")
	cmd.Flags().StringVar(&project, "project", "", "DC project key")
	cmd.Flags().BoolVar(&setActive, "set-active", false, "make this the active context")
	return cmd
}

func newContextDeleteCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"remove", "rm"},
		Short:   "Delete a context and its stored credentials",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			c := cfg.Find(args[0])
			if c == nil {
				return fmt.Errorf("context %q not found", args[0])
			}
			if !cfg.Delete(args[0]) {
				return fmt.Errorf("failed to delete context")
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			// Best-effort credential removal.
			_ = auth.Delete(c)
			stderrf("Deleted context %q", args[0])
			return nil
		},
	}
}
