package cli

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/browser"
	"github.com/spf13/cobra"

	"github.com/arogan178/bitbucket-cli/internal/auth"
	"github.com/arogan178/bitbucket-cli/internal/bitbucket"
	"github.com/arogan178/bitbucket-cli/internal/config"
)

const (
	cloudTokenManageURL      = "https://id.atlassian.com/manage-profile/security/api-tokens"
	cloudTokenCreateDocsURL  = "https://support.atlassian.com/bitbucket-cloud/docs/create-an-api-token/"
	cloudTokenPermsDocsURL   = "https://support.atlassian.com/bitbucket-cloud/docs/api-token-permissions/"
	cloudAppPasswordSunset   = "2026-06-09"
	dataCenterTokenDocsURL   = "https://confluence.atlassian.com/bitbucketserver/managing-personal-access-tokens-1005339986.html"
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
		host              string
		kind              string
		contextName       string
		username          string
		token             string
		workspace         string
		project           string
		useWeb            bool
		noWeb             bool
		setActive         bool
		skipValidate      bool
		legacyAppPassword bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Guided sign-in for Bitbucket Cloud or Data Center",
		Long: `Log in to Bitbucket Cloud or a Data Center host.

Examples:

  # Bitbucket Cloud (Atlassian account email + API token)
  bt auth login --kind cloud --username alice@example.com --token <API token> \
       --workspace myteam

  # Bitbucket Data Center (PAT)
  bt auth login --host https://bitbucket.example.com --kind dc \
       --username alice --token <PAT> --project ABC --context dc-prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if useWeb && noWeb {
				return fmt.Errorf("--web and --no-web are mutually exclusive")
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			interactive := isTerminalFile(os.Stdin)
			if kind == "" {
				if interactive {
					kind = promptWithDefault("Bitbucket kind", "cloud")
				} else {
					kind = "cloud"
				}
			}
			kind = strings.ToLower(strings.TrimSpace(kind))
			k := config.Kind(kind)
			switch k {
			case config.KindCloud, config.KindDataCenter:
			default:
				return fmt.Errorf("invalid --kind %q; expected cloud or dc", kind)
			}

			if host == "" {
				if k == config.KindCloud {
					host = "https://bitbucket.org"
				} else if interactive {
					host = prompt("Bitbucket Data Center host (e.g. https://bitbucket.example.com): ")
				} else {
					return fmt.Errorf("--host is required for Data Center logins")
				}
			}
			host = normalizeHost(host)

			if contextName == "" {
				contextName = defaultContextName(host, k)
			}

			var mode string
			switch k {
			case config.KindCloud:
				if legacyAppPassword {
					stderrf("Legacy Bitbucket Cloud app-password mode selected.")
					stderrf("App passwords are deprecated and stop working on %s.", cloudAppPasswordSunset)
				}
				if username == "" {
					label := "Atlassian account email: "
					if legacyAppPassword {
						label = "Bitbucket username (legacy app password): "
					}
					username = prompt(label)
				}
				if workspace == "" && interactive {
					workspace = prompt("Default workspace (optional): ")
				}
				if token == "" {
					if interactive && !noWeb && (useWeb || confirm("Open browser to create/manage a Bitbucket Cloud token now?", true)) {
						openCloudLoginHelp(legacyAppPassword)
					}
					if legacyAppPassword {
						token = promptSecret("Paste Bitbucket Cloud app password: ")
					} else {
						token = promptSecret("Paste Bitbucket Cloud API token: ")
					}
				}
				mode = "api_token"
				if legacyAppPassword {
					mode = "app_password"
				}
				if mode == "api_token" && !strings.Contains(username, "@") {
					return fmt.Errorf("Bitbucket Cloud API tokens require your Atlassian account email as --username (not your Bitbucket nickname)")
				}
			case config.KindDataCenter:
				if username == "" {
					username = prompt("Bitbucket username: ")
				}
				if project == "" && interactive {
					project = prompt("Default project key (optional): ")
				}
				if token == "" {
					if interactive && !noWeb && (useWeb || confirm("Open browser to your Bitbucket Data Center token settings now?", true)) {
						openDataCenterLoginHelp(host)
					}
					token = promptSecret("Paste Bitbucket Data Center personal access token: ")
				}
				mode = "pat"
			}

			username = strings.TrimSpace(username)
			token = strings.TrimSpace(token)
			workspace = strings.TrimSpace(workspace)
			project = strings.TrimSpace(project)

			if username == "" {
				return fmt.Errorf("no username/email provided")
			}
			if token == "" {
				return fmt.Errorf("no token provided")
			}

			ctx := config.Context{
				Name:      contextName,
				Host:      host,
				Kind:      k,
				Workspace: workspace,
				Project:   project,
				Username:  username,
			}
			cred := auth.Credential{
				Kind:      k,
				Principal: username,
				Secret:    token,
				Mode:      mode,
			}

			loggedInAs := username
			if !skipValidate {
				who, err := validateLogin(&ctx, cred)
				if err != nil {
					return err
				}
				if who != "" {
					loggedInAs = who
				}
				stderrf("Authenticated as %s", loggedInAs)
			}

			cfg.Upsert(ctx)
			if setActive || cfg.Active == "" {
				cfg.Active = contextName
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			if err := auth.Store(&ctx, cred); err != nil {
				return err
			}
			stderrf("Logged in to %s as %s (context %q)", host, loggedInAs, contextName)
			if k == config.KindCloud && workspace == "" {
				stderrf("Tip: no default workspace set. Pass --workspace now or set one later with `bt context`.")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&host, "host", "", "Bitbucket host (default https://bitbucket.org)")
	cmd.Flags().StringVar(&kind, "kind", "", "cloud | dc (default cloud)")
	cmd.Flags().StringVar(&contextName, "context", "", "context name (defaults based on host)")
	cmd.Flags().StringVar(&username, "username", "", "Atlassian account email (Cloud API token) or Bitbucket username (DC / legacy app password)")
	cmd.Flags().StringVar(&token, "token", "", "Cloud API token, legacy app password, or Data Center PAT")
	cmd.Flags().StringVar(&workspace, "workspace", "", "Cloud workspace (recommended for repo/pr list)")
	cmd.Flags().StringVar(&project, "project", "", "Data Center project key")
	cmd.Flags().BoolVar(&useWeb, "web", false, "open the browser immediately to the token creation page")
	cmd.Flags().BoolVar(&noWeb, "no-web", false, "never open a browser during guided login")
	cmd.Flags().BoolVar(&skipValidate, "skip-validate", false, "store credentials without making a test API call first")
	cmd.Flags().BoolVar(&legacyAppPassword, "legacy-app-password", false, "Bitbucket Cloud only: use a deprecated app password instead of an API token")
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

func validateLogin(ctx *config.Context, cred auth.Credential) (string, error) {
	cl, err := bitbucket.New(ctx, cred)
	if err != nil {
		return "", err
	}

	switch ctx.Kind {
	case config.KindCloud:
		data, err := cl.Raw().Do(withCtx(), "GET", "/2.0/user", nil, nil)
		if err != nil {
			return "", explainValidationError(ctx, cred, err)
		}
		var user struct {
			DisplayName string `json:"display_name"`
			Nickname    string `json:"nickname"`
			Username    string `json:"username"`
		}
		if err := json.Unmarshal(data, &user); err != nil {
			return "", err
		}
		switch {
		case user.DisplayName != "":
			return user.DisplayName, nil
		case user.Nickname != "":
			return user.Nickname, nil
		case user.Username != "":
			return user.Username, nil
		default:
			return ctx.Username, nil
		}
	case config.KindDataCenter:
		if _, err := cl.Raw().Do(withCtx(), "GET", "/rest/api/1.0/projects", map[string]string{"limit": "1"}, nil); err != nil {
			return "", explainValidationError(ctx, cred, err)
		}
		return ctx.Username, nil
	default:
		return ctx.Username, nil
	}
}

func explainValidationError(ctx *config.Context, cred auth.Credential, err error) error {
	var apiErr *bitbucket.APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	switch ctx.Kind {
	case config.KindCloud:
		if cred.Mode == "app_password" {
			return fmt.Errorf("Bitbucket Cloud rejected the app password (%w). App passwords are deprecated and stop working on %s; prefer API tokens from %s", apiErr, cloudAppPasswordSunset, cloudTokenManageURL)
		}
		return fmt.Errorf("Bitbucket Cloud rejected the credentials (%w). API tokens require your Atlassian account email as --username. Create/manage tokens at %s", apiErr, cloudTokenManageURL)
	case config.KindDataCenter:
		return fmt.Errorf("Bitbucket Data Center rejected the personal access token (%w). Create a PAT under Manage account > Personal access tokens. Docs: %s", apiErr, dataCenterTokenDocsURL)
	default:
		return err
	}
}

func openCloudLoginHelp(legacyAppPassword bool) {
	if legacyAppPassword {
		stderrf("Legacy app-password mode does not open a creation page because Bitbucket Cloud no longer allows creating new app passwords.")
		stderrf("If you do not already have one, use the default API token flow instead: %s", cloudTokenManageURL)
		return
	}
	openURL(cloudTokenManageURL)
	stderrf("Bitbucket Cloud API tokens are now the recommended login method.")
	stderrf("App passwords are deprecated and stop working on %s.", cloudAppPasswordSunset)
	stderrf("Suggested read-only scopes: User Read, Workspace Read, Repository Read, Pull request Read.")
	stderrf("Suggested extra scopes for full bt usage: Repository Write/Admin/Delete, Pull request Write, Pipeline Read/Write, Issue Read/Write/Delete, Webhook Read/Write/Delete.")
	stderrf("Docs: %s", cloudTokenCreateDocsURL)
	stderrf("Permissions reference: %s", cloudTokenPermsDocsURL)
}

func openDataCenterLoginHelp(host string) {
	openURL(host)
	stderrf("In Bitbucket Data Center, create a Personal access token under Manage account > Personal access tokens.")
	stderrf("Docs: %s", dataCenterTokenDocsURL)
}

func openURL(target string) {
	if err := browser.OpenURL(target); err != nil {
		stderrf("Couldn't open a browser automatically: %v", err)
		stderrf("Open this URL manually: %s", target)
		return
	}
	stderrf("Opened %s", target)
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimRight(host, "/")
	if host == "" {
		return host
	}
	if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
		host = "https://" + host
	}
	return host
}

func promptWithDefault(label, defaultValue string) string {
	if defaultValue == "" {
		return prompt(label + ": ")
	}
	answer := prompt(fmt.Sprintf("%s [%s]: ", label, defaultValue))
	if answer == "" {
		return defaultValue
	}
	return answer
}

func confirm(label string, defaultYes bool) bool {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	answer := strings.ToLower(prompt(fmt.Sprintf("%s %s: ", label, suffix)))
	if answer == "" {
		return defaultYes
	}
	return answer == "y" || answer == "yes"
}

func isTerminalFile(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
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
