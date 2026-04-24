package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

func newWebhookCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "webhook",
		Aliases: []string{"hook"},
		Short:   "Manage repository webhooks",
	}
	cmd.AddCommand(newWebhookListCmd(g), newWebhookCreateCmd(g), newWebhookDeleteCmd(g))
	return cmd
}

func newWebhookListCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List webhooks for a repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			hooks, err := cl.Webhooks().List(withCtx(), slug)
			if err != nil {
				return err
			}
			cols := []string{"ID", "NAME", "ACTIVE", "URL", "EVENTS"}
			rows := make([][]string, 0, len(hooks))
			for _, h := range hooks {
				active := "no"
				if h.Active {
					active = "yes"
				}
				rows = append(rows, []string{h.ID, h.Name, active, h.URL, strings.Join(h.Events, ",")})
			}
			return renderWith(g, hooks, cols, rows)
		},
	}
}

func newWebhookCreateCmd(g *GlobalFlags) *cobra.Command {
	var name, url string
	var events []string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			slug, err := g.repoSlug("")
			if err != nil {
				return err
			}
			w, err := cl.Webhooks().Create(withCtx(), slug, name, url, events)
			if err != nil {
				return err
			}
			return renderValueWith(g, w)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "webhook name/description")
	cmd.Flags().StringVar(&url, "url", "", "webhook URL")
	cmd.Flags().StringSliceVar(&events, "event", nil, "event to subscribe to (repeatable)")
	_ = cmd.MarkFlagRequired("url")
	return cmd
}

func newWebhookDeleteCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"rm"},
		Short:   "Delete a webhook",
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
			return cl.Webhooks().Delete(withCtx(), slug, args[0])
		},
	}
}
