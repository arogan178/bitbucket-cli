package cli

import "github.com/arogan178/bitbucket-cli/internal/output"

// output_Render is a module-private helper so every subcommand renders
// through the same options builder without repeating boilerplate.
func output_Render(g *GlobalFlags, value any, cols []string, rows [][]string) error {
	return output.New(g.outputOpts()).Render(value, cols, rows)
}

func output_RenderValue(g *GlobalFlags, value any) error {
	return output.New(g.outputOpts()).RenderValue(value)
}
