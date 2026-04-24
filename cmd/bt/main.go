// Command bt is a gh-style Bitbucket CLI that speaks both Bitbucket
// Cloud (api.bitbucket.org/2.0) and Bitbucket Data Center
// (rest/api/1.0) via a single static binary.
package main

import (
	"fmt"
	"os"

	"github.com/arogan178/bitbucket-cli/internal/cli"
)

// Version is overridden at build time by goreleaser via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	root := cli.NewRootCmd(cli.BuildInfo{
		Version: Version,
		Commit:  Commit,
		Date:    Date,
	})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
