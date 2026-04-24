package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newAPICmd(g *GlobalFlags) *cobra.Command {
	var method string
	var params []string
	var body string
	var stdin bool
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make an authenticated raw API request",
		Long: `Make an arbitrary authenticated request against the backend that matches the
active context. Path is relative to the host:

  # Bitbucket Cloud
  bt api /2.0/repositories/myteam --param role=member

  # Bitbucket Data Center
  bt api /rest/api/1.0/projects --param limit=100

Anything sent via --body is POSTed as JSON. Use --stdin to stream stdin as
the request body (useful with heredocs).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, _, err := g.client()
			if err != nil {
				return err
			}
			kv := map[string]string{}
			for _, p := range params {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) == 2 {
					kv[parts[0]] = parts[1]
				}
			}
			var reader io.Reader
			if stdin {
				reader = os.Stdin
			} else if body != "" {
				reader = bytes.NewReader([]byte(body))
			}
			data, err := cl.Raw().Do(withCtx(), strings.ToUpper(method), args[0], kv, reader)
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}
	cmd.Flags().StringVarP(&method, "method", "X", "GET", "HTTP method")
	cmd.Flags().StringSliceVar(&params, "param", nil, "KEY=VALUE query string param (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "request body (JSON string)")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read request body from stdin")
	return cmd
}
