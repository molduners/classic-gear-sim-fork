package core

import (
	"fmt"
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

// Mechanic tests for a level 60 player meleeing a level 63 boss in vanilla
// Classic Era. Expected values come from community testing on the Classic
// clients, not from this code base:
//
//   [magey] magey et al., "Attack table", classic-warrior wiki (2019):
//           https://github.com/magey/classic-warrior/wiki/Attack-table
//           Boss-dummy tests: dodge 6.43% ±0.22% (n=48,445), parry 13.97% ±0.38%,
//           glancing 40.36% ±0.54% with ~35% damage reduction (n=31,779),
//           dual-wield miss = normal miss + 19% (n=48,445).
//   [marrow] Marrow's Classic Warrior Compendium, ch. 3 "Mechanics":
//           https://bookdown.org/marrowwar/marrow_compendium/mechanics.html
//           Table 3.2: miss and glancing damage by weapon skill.
//   [rage]  Rage conversion C = 0.0091107836*L^2 + 3.225598133*L + 4.2652911
//           (230.6 at level 60). Marrow's compendium ch. 4, table 4.1:
//           https://bookdown.org/marrowwar/marrow_compendium/abilities.html

func vanillaAttackTable(bonusSkill float64) *AttackTable {
	attacker := &Unit{Type: PlayerUnit, Level: 60, PseudoStats: stats.NewPseudoStats()}
	attacker.PseudoStats.TwoHandedSwordsSkill = bonusSkill
	defender := &Unit{Type: EnemyUnit, Level: 63, PseudoStats: stats.NewPseudoStats()}
	weapon := &Item{HandType: proto.HandType_HandTypeTwoHand, WeaponType: proto.WeaponType_WeaponTypeSword}
	return NewAttackTable(attacker, defender, weapon)
}

func assertClose(t *testing.T, name string, want, got float64) {
	t.Helper()
	if math.Abs(want-got) > 1e-9 {
		t.Errorf("%s: want %.4f, got %.4f", name, want, got)
	}
}

func TestVanillaAttackTableVsBoss(t *testing.T) {
	cases := []struct {
		skill        float64 // total weapon skill
		miss         float64 // white miss, single weapon [marrow table 3.2]
		dodge        float64 // [magey]: 5% + (315 - skill) * 0.1%
		glanceDamage float64 // average glancing damage multiplier [marrow table 3.2]
	}{
		{300, 0.080, 0.065, 0.65},
		{304, 0.072, 0.061, 0.81},
		{305, 0.060, 0.060, 0.85},
		{308, 0.057, 0.057, 0.95},
		{315, 0.050, 0.050, 0.95},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("skill%.0f", c.skill), func(t *testing.T) {
			at := vanillaAttackTable(c.skill - 300)
			assertClose(t, "miss", c.miss, at.BaseMissChance)
			assertClose(t, "dodge", c.dodge, at.BaseDodgeChance)
			// Glancing chance does not depend on weapon skill above 300 [magey].
			assertClose(t, "glance chance", 0.40, at.BaseGlanceChance)
			// Parry from the front is a flat 14% vs +3 [magey]; it is not reduced by skill in the sim.
			assertClose(t, "parry", 0.14, at.BaseParryChance)
			assertClose(t, "glance damage", c.glanceDamage, (at.GlanceMultiplierMin+at.GlanceMultiplierMax)/2)
			// Total crit suppression vs +3 is 4.8%: 3% from level, 1.8% from auras [magey].
			assertClose(t, "crit suppression", 0.048, at.MeleeCritSuppression)
		})
	}
}

// The hit cap at 300 skill is 9%, one point above the 8% miss chance, because
// of hit suppression [marrow table 3.2]. From 305 skill on there is none.
func TestVanillaHitSuppression(t *testing.T) {
	assertClose(t, "suppression at 300", 0.01, vanillaAttackTable(0).HitSuppression)
	assertClose(t, "suppression at 305", 0.0, vanillaAttackTable(5).HitSuppression)
}

func TestVanillaRageConversion(t *testing.T) {
	// 0.0091107836*3600 + 3.225598133*60 + 4.2652911 = 230.6 [rage]
	got := GetRageConversion(60)
	if math.Abs(got-230.6) > 0.05 {
		t.Fatalf("rage conversion at level 60: want 230.6, got %.2f", got)
	}
	// Tested values from the same source at levels 25 and 40.
	assertClose(t, "level 25", 82.25, GetRageConversion(25))
	assertClose(t, "level 40", 140.5, GetRageConversion(40))
}

// Physical damage reduction from armor for a level 60 attacker is
// armor / (armor + 400 + 85*60), capped at 75%.
// Source: https://wow.allakhazam.com/wiki/Armor_(WoW) gives 400 + 85*level and the
// 75% cap. Its separate "60+" formula is the TBC one; Classic Era keeps 400 + 85*level
// at 60, as do the community warrior sims.
func TestVanillaArmorReduction(t *testing.T) {
	for _, c := range []struct{ armor, reduction float64 }{
		{0, 0},
		{3731, 3731.0 / (3731.0 + 5500.0)}, // typical raid boss armor, ~40.4%
		{100000, 0.75},
	} {
		attacker := &Unit{Type: PlayerUnit, Level: 60, PseudoStats: stats.NewPseudoStats()}
		defender := &Unit{Type: EnemyUnit, Level: 63, PseudoStats: stats.NewPseudoStats()}
		defender.stats[stats.Armor] = c.armor
		at := &AttackTable{Attacker: attacker, Defender: defender}
		assertClose(t, fmt.Sprintf("armor %.0f", c.armor), 1-c.reduction, at.GetArmorDamageModifier())
	}
}
