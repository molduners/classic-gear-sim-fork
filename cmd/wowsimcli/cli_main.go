package main

import (
	"github.com/wowsims/classic/cmd/wowsimcli/cmd"

	// Each supported game version registers itself as a ruleset.
	_ "github.com/wowsims/classic/sim/ruleset/vanilla"
)

// Version information.
// This variable is set by the makefile in the release process.
var Version string

func main() {
	cmd.Execute(Version)
}
