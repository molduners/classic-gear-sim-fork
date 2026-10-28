// Package vanilla is the ruleset for World of Warcraft Classic Era (patch 1.15,
// level 60, the original nine classes).
package vanilla

import (
	_ "github.com/wowsims/classic/sim/common"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/ruleset"

	"github.com/wowsims/classic/sim/druid/balance"
	"github.com/wowsims/classic/sim/druid/feral"
	// restoDruid "github.com/wowsims/classic/sim/druid/restoration"
	// feralTank "github.com/wowsims/classic/sim/druid/tank"
	_ "github.com/wowsims/classic/sim/encounters"
	"github.com/wowsims/classic/sim/hunter"
	"github.com/wowsims/classic/sim/mage"

	// holyPaladin "github.com/wowsims/classic/sim/paladin/holy"
	"github.com/wowsims/classic/sim/paladin/protection"
	"github.com/wowsims/classic/sim/paladin/retribution"

	healingPriest "github.com/wowsims/classic/sim/priest/healing"
	"github.com/wowsims/classic/sim/priest/shadow"
	dpsrogue "github.com/wowsims/classic/sim/rogue/dps_rogue"

	"github.com/wowsims/classic/sim/shaman/elemental"
	"github.com/wowsims/classic/sim/shaman/enhancement"

	// restoShaman "github.com/wowsims/classic/sim/shaman/restoration"
	dpsWarlock "github.com/wowsims/classic/sim/warlock/dps"
	dpsWarrior "github.com/wowsims/classic/sim/warrior/dps_warrior"
	tankWarrior "github.com/wowsims/classic/sim/warrior/tank_warrior"
)

const ID = "vanilla"

func init() {
	ruleset.Register(&ruleset.Ruleset{
		ID:            ID,
		Name:          "Classic Era (vanilla)",
		MaxLevel:      core.CharacterMaxLevel,
		RegisterSpecs: registerSpecs,
	})
}

func registerSpecs() {
	balance.RegisterBalanceDruid()
	feral.RegisterFeralDruid()
	// feralTank.RegisterFeralTankDruid()
	// restoDruid.RegisterRestorationDruid()
	elemental.RegisterElementalShaman()
	enhancement.RegisterEnhancementShaman()
	// restoShaman.RegisterRestorationShaman()
	hunter.RegisterHunter()
	mage.RegisterMage()
	healingPriest.RegisterHealingPriest()
	shadow.RegisterShadowPriest()
	dpsrogue.RegisterDpsRogue()
	dpsWarrior.RegisterDpsWarrior()
	tankWarrior.RegisterTankWarrior()
	// holyPaladin.RegisterHolyPaladin()
	protection.RegisterProtectionPaladin()
	retribution.RegisterRetributionPaladin()
	dpsWarlock.RegisterDpsWarlock()
}
