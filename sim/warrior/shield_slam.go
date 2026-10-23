package warrior

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

func (warrior *Warrior) registerShieldSlamSpell() {
	if !warrior.Talents.ShieldSlam {
		return
	}

	// Rank 4. Damage from the Era tooltip: "causing 342 to 358 damage, modified by your shield
	// block value" (https://www.wowhead.com/classic/spell=23925). Season of Discovery changed this
	// to twice the block value and added attack power; Era adds block value once and has no AP scaling.
	// UNCERTAIN: no in-game test of the block value part; the tooltip wording is the only source.
	spellID := int32(23925)
	damageLow := 342.0
	damageHigh := 358.0
	// Bonus threat measured in game on 1.13.6: https://github.com/magey/classic-warrior/wiki/Threat-Mechanics
	threat := 254.0

	warrior.ShieldSlam = warrior.RegisterSpell(AnyStance, core.SpellConfig{
		SpellCode:   SpellCode_WarriorShieldSlam,
		ActionID:    core.ActionID{SpellID: spellID},
		SpellSchool: core.SpellSchoolPhysical,
		DefenseType: core.DefenseTypeMelee,
		ProcMask:    core.ProcMaskMeleeMHSpecial, // TODO really?
		Flags:       core.SpellFlagMeleeMetrics | core.SpellFlagAPL | SpellFlagOffensive,

		RageCost: core.RageCostOptions{
			Cost:   20,
			Refund: 0.8,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD: core.GCDDefault,
			},
			IgnoreHaste: true,
			CD: core.Cooldown{
				Timer:    warrior.NewTimer(),
				Duration: time.Second * 6,
			},
		},
		ExtraCastCondition: func(sim *core.Simulation, target *core.Unit) bool {
			return warrior.PseudoStats.CanBlock
		},

		CritDamageBonus: warrior.impale(),

		DamageMultiplier: 1,
		ThreatMultiplier: 1,
		FlatThreatBonus:  threat,
		BonusCoefficient: 1,

		ApplyEffects: func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			damage := sim.Roll(damageLow, damageHigh) + warrior.BlockValue()
			result := spell.CalcAndDealDamage(sim, target, damage, spell.OutcomeMeleeSpecialHitAndCrit)

			if !result.Landed() {
				spell.IssueRefund(sim)
			}
		},
	})
}
