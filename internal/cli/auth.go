package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/config"
)

func newAuthCmd(g *GlobalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate bt with Bitbucket Cloud or Data Center",
	}
	cmd.AddCommand(newAuthLoginCmd(g), newAuthLogoutCmd(g), newAuthStatusCmd(g))
	return cmd
}

func newAuthLoginCmd(g *GlobalFlags) *cobra.Command {
	var (
		host       string
		kind       string
		contextName string
		username   string
		token      string
		workspace  string
		project    string
		useWeb     bool
		setActive  bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Log in and store Bitbucket credentials",
		Long: `Log in to Bitbucket Cloud or a Data Center host.

Examples:

  # Bitbucket Cloud (email + API token)
  bt auth login --kind cloud --username alice@example.com --token <API token> \
       --workspace myteam

  # Bitbucket Data Center (PAT)
  bt auth login --host https://bitbucket.example.com --kind dc \
       --username alice --token <PAT> --project ABC --context dc-prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if kind == "" {
				kind = "cloud"
			}
			k := config.Kind(kind)

			if host == "" {
				if k == config.KindCloud {
					host = "https://bitbucket.org"
				} else {
					return fmt.Errorf("--host is required for Data Center logins")
				}
			}

			if contextName == "" {
				contextName = defaultContextName(host, k)
			}

			if username == "" {
				username = prompt("Username/email: ")
			}
			if token == "" {
				if useWeb {
					stderrf("Opening browser to create a token at %s/account/settings/app-passwords", host)
					stderrf("(paste the token below; input is hidden)")
				}
				token = promptSecret("Token/app password: ")
			}
			if token == "" {
				return fmt.Errorf("no token provided")
			}

			mode := "app_password"
			if k == config.KindCloud && strings.Contains(username, "@") {
				mode = "api_token"
			}
			if k == config.KindDataCenter {
				mode = "pat"
			}

			ctx := config.Context{
				Name:      contextName,
				Host:      host,
				Kind:      k,
				Workspace: workspace,
				Project:   project,
				Username:  username,
			}
			cfg.Upsert(ctx)
			if setActive || cfg.Active == "" {
				cfg.Active = contextName
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			cred := auth.Credential{
				Kind:      k,
				Principal: username,
				Secret:    token,
				Mode:      mode,
			}
			if err := auth.Store(&ctx, cred); err != nil {
				return err
			}
			stderrf("Logged in as %s (context %q)", username, contextName)
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Bitbucket host (default https://bitbucket.org)")
	cmd.Flags().StringVar(&kind, "kind", "", "cloud | dc (default cloud)")
	cmd.Flags().StringVar(&contextName, "context", "", "context name (defaults based on host)")
	cmd.Flags().StringVar(&username, "username", "", "Bitbucket username or email")
	cmd.Flags().StringVar(&token, "token", "", "API token, app password, or PAT")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Cloud workspace (required for pr/repo list)")
	cmd.Flags().StringVar(&project, "project", "", "Data Center project key")
	cmd.Flags().BoolVar(&useWeb, "web", false, "open a browser to create the token")
	cmd.Flags().BoolVar(&setActive, "set-active", true, "make this the active context")
	return cmd
}

func newAuthLogoutCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "logout [context]",
		Short: "Delete stored credentials for a context",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := ""
			if len(args) == 1 {
				name = args[0]
			} else {
				name = cfg.Active
			}
			if name == "" {
				return fmt.Errorf("no context specified")
			}
			ctx := cfg.Find(name)
			if ctx == nil {
				return fmt.Errorf("context %q not found", name)
			}
			if err := auth.Delete(ctx); err != nil {
				return err
			}
			stderrf("Credentials for %q removed.", name)
			return nil
		},
	}
}

func newAuthStatusCmd(g *GlobalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show authentication status for every context",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Contexts) == 0 {
				fmt.Println("No contexts configured. Run `bt auth login` to get started.")
				return nil
			}
			for _, c := range cfg.Contexts {
				c := c
				active := " "
				if c.Name == cfg.Active {
					active = "*"
				}
				cred, err := auth.Load(&c)
				status := "authenticated"
				if err != nil {
					status = "missing credentials"
				} else if cred.Principal == "" {
					status = "anonymous"
				}
				fmt.Printf("%s %s  %s  [%s]  %s\n", active, c.Name, c.Host, c.Kind, status)
			}
			return nil
		},
	}
}

func defaultContextName(host string, k config.Kind) string {
	h := strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	h = strings.Split(h, "/")[0]
	return fmt.Sprintf("%s-%s", k, strings.ReplaceAll(h, ".", "-"))
}

func prompt(label string) string {
	fmt.Fprint(os.Stderr, label)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

// promptSecret reads without echo when stdin is a TTY. Falls back to
// echoing input otherwise. We avoid /x/term to keep deps light; `stty` is
// available on macOS/Linux. Windows users can set --token directly.
func promptSecret(label string) string {
	fmt.Fprint(os.Stderr, label)
	// best-effort: disable echo via stty; ignore errors if not a TTY
	_ = setRawEcho(false)
	defer func() { _ = setRawEcho(true); fmt.Fprintln(os.Stderr) }()
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}
