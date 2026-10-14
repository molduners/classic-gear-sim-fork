package sim

import (
	"github.com/wowsims/classic/sim/ruleset"

	// Each supported game version registers itself as a ruleset.
	_ "github.com/wowsims/classic/sim/ruleset/vanilla"
)

// RegisterAll activates the default ruleset. Entry points that let the user
// choose a version (the CLI's --ruleset flag) call ruleset.Activate instead.
func RegisterAll() {
	if err := ruleset.Activate(ruleset.Default); err != nil {
		panic(err)
	}
}
