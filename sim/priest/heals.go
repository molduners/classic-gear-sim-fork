package priest

import (
	"time"

	"github.com/wowsims/classic/sim/core"
)

// Vanilla (1.12) Holy Priest heals.
//
// Rank data comes from the 1.12.1 spell DBC (cmangos classic-db spell_template):
// healing = EffectBasePoints+1 .. EffectBasePoints+EffectDieSides, plus
// EffectRealPointsPerLevel for each level the caster is above the spell's level
// (capped at the spell's MaxLevel), as in vmangos SpellCaster::CalculateSpellEffectValue.
//
// +healing coefficients use the 1.12 server default (vmangos/cmangos CalculateDefaultCoefficient):
// cast time / 3.5s for direct heals, duration / 15s for HoTs, and a further
// 1 - (20 - spellLevel) * 0.0375 penalty for spells learned below level 20.
type HealRankData struct {
	SpellID  int32
	Level    int32
	MaxLevel int32
	Mana     float64
	CastMs   int
	Min      float64 // per tick for HoTs
	Max      float64
	PerLevel float64
}

var HealRanks = []HealRankData{
	{SpellID: 2054, Level: 16, MaxLevel: 21, Mana: 155, CastMs: 3000, Min: 295, Max: 341, PerLevel: 2.4},
	{SpellID: 2055, Level: 22, MaxLevel: 27, Mana: 205, CastMs: 3000, Min: 429, Max: 491, PerLevel: 3.2},
	{SpellID: 6063, Level: 28, MaxLevel: 33, Mana: 255, CastMs: 3000, Min: 566, Max: 642, PerLevel: 4},
	{SpellID: 6064, Level: 34, MaxLevel: 39, Mana: 305, CastMs: 3000, Min: 712, Max: 804, PerLevel: 4.5},
}

var GreaterHealRanks = []HealRankData{
	{SpellID: 2060, Level: 40, MaxLevel: 45, Mana: 370, CastMs: 3000, Min: 899, Max: 1013, PerLevel: 5.1},
	{SpellID: 10963, Level: 46, MaxLevel: 51, Mana: 455, CastMs: 3000, Min: 1149, Max: 1289, PerLevel: 5.8},
	{SpellID: 10964, Level: 52, MaxLevel: 57, Mana: 545, CastMs: 3000, Min: 1437, Max: 1609, PerLevel: 6.6},
	{SpellID: 10965, Level: 58, MaxLevel: 63, Mana: 655, CastMs: 3000, Min: 1798, Max: 2006, PerLevel: 7.5},
	{SpellID: 25314, Level: 60, MaxLevel: 65, Mana: 710, CastMs: 3000, Min: 1966, Max: 2194, PerLevel: 8.1},
}

var FlashHealRanks = []HealRankData{
	{SpellID: 2061, Level: 20, MaxLevel: 25, Mana: 125, CastMs: 1500, Min: 193, Max: 237, PerLevel: 1.9},
	{SpellID: 9472, Level: 26, MaxLevel: 31, Mana: 155, CastMs: 1500, Min: 258, Max: 314, PerLevel: 2.2},
	{SpellID: 9473, Level: 32, MaxLevel: 37, Mana: 185, CastMs: 1500, Min: 327, Max: 393, PerLevel: 2.5},
	{SpellID: 9474, Level: 38, MaxLevel: 43, Mana: 215, CastMs: 1500, Min: 400, Max: 478, PerLevel: 2.8},
	{SpellID: 10915, Level: 44, MaxLevel: 49, Mana: 265, CastMs: 1500, Min: 518, Max: 616, PerLevel: 3.3},
	{SpellID: 10916, Level: 50, MaxLevel: 55, Mana: 315, CastMs: 1500, Min: 644, Max: 764, PerLevel: 3.7},
	{SpellID: 10917, Level: 56, MaxLevel: 61, Mana: 380, CastMs: 1500, Min: 812, Max: 958, PerLevel: 4.2},
}

// Renew: instant, 15s, a tick every 3s. Min/Max is the amount per tick.
var RenewRanks = []HealRankData{
	{SpellID: 139, Level: 8, MaxLevel: 13, Mana: 30, Min: 9, Max: 9},
	{SpellID: 6074, Level: 14, MaxLevel: 19, Mana: 65, Min: 20, Max: 20},
	{SpellID: 6075, Level: 20, MaxLevel: 25, Mana: 105, Min: 35, Max: 35},
	{SpellID: 6076, Level: 26, MaxLevel: 31, Mana: 140, Min: 49, Max: 49},
	{SpellID: 6077, Level: 32, MaxLevel: 37, Mana: 170, Min: 63, Max: 63},
	{SpellID: 6078, Level: 38, MaxLevel: 43, Mana: 205, Min: 80, Max: 80},
	{SpellID: 10927, Level: 44, MaxLevel: 49, Mana: 250, Min: 102, Max: 102},
	{SpellID: 10928, Level: 50, MaxLevel: 55, Mana: 305, Min: 130, Max: 130},
	{SpellID: 10929, Level: 56, MaxLevel: 61, Mana: 365, Min: 162, Max: 162},
	{SpellID: 25315, Level: 60, MaxLevel: 65, Mana: 410, Min: 194, Max: 194},
}

var PrayerOfHealingRanks = []HealRankData{
	{SpellID: 596, Level: 30, MaxLevel: 39, Mana: 410, CastMs: 3000, Min: 301, Max: 321, PerLevel: 1.3},
	{SpellID: 996, Level: 40, MaxLevel: 49, Mana: 560, CastMs: 3000, Min: 444, Max: 472, PerLevel: 1.6},
	{SpellID: 10960, Level: 50, MaxLevel: 59, Mana: 770, CastMs: 3000, Min: 657, Max: 695, PerLevel: 2},
	{SpellID: 10961, Level: 60, MaxLevel: 69, Mana: 1030, CastMs: 3000, Min: 939, Max: 991, PerLevel: 2.4},
	{SpellID: 25316, Level: 60, MaxLevel: 69, Mana: 1070, CastMs: 3000, Min: 1041, Max: 1099, PerLevel: 2.5},
}

const (
	RenewDuration = time.Second * 15
	RenewTicks    = 5

	// UNCERTAIN: the server default would be 3/3.5 per target. Community
	// tables for 1.12 (e.g. us.forums.blizzard.com/t/319852) give 0.286,
	// i.e. a third of that, which matches how much +healing PoH gets in Era.
	PrayerOfHealingCoefficient = 3.0 / 3.5 / 3
)

// Healing added by caster levels above the spell's level, capped at MaxLevel.
func (rank HealRankData) levelBonus(casterLevel int32) float64 {
	levels := min(casterLevel, rank.MaxLevel) - rank.Level
	return max(0, float64(levels)) * rank.PerLevel
}

// Spells learned below level 20 get less from +healing.
func (rank HealRankData) levelPenalty() float64 {
	if rank.Level >= 20 {
		return 1
	}
	return 1 - float64(20-rank.Level)*0.0375
}

func (rank HealRankData) directCoefficient() float64 {
	return min(1, float64(rank.CastMs)/3500) * rank.levelPenalty()
}

func (priest *Priest) RegisterHealingSpells() {
	priest.Heal = priest.registerDirectHeal(HealRanks, SpellCode_PriestHeal, true)
	priest.GreaterHeal = priest.registerDirectHeal(GreaterHealRanks, SpellCode_PriestGreaterHeal, true)
	priest.FlashHeal = priest.registerDirectHeal(FlashHealRanks, SpellCode_PriestFlashHeal, false)
	priest.registerRenew()
	priest.registerPrayerOfHealing()
}

func (priest *Priest) healSpellConfig(rank HealRankData, rankNum int, spellCode int32) core.SpellConfig {
	return core.SpellConfig{
		ActionID:    core.ActionID{SpellID: rank.SpellID},
		SpellCode:   spellCode,
		SpellSchool: core.SpellSchoolHoly,
		DefenseType: core.DefenseTypeMagic,
		ProcMask:    core.ProcMaskSpellHealing,
		Flags:       SpellFlagPriest | core.SpellFlagHelpful | core.SpellFlagAPL,

		RequiredLevel: int(rank.Level),
		Rank:          rankNum,

		ManaCost: core.ManaCostOptions{
			FlatCost: rank.Mana,
		},
		Cast: core.CastConfig{
			DefaultCast: core.Cast{
				GCD:      core.GCDDefault,
				CastTime: time.Millisecond * time.Duration(rank.CastMs),
			},
		},

		// Spiritual Healing: +2% healing per rank (spellmod on the whole heal, base and bonus).
		DamageMultiplier:         1,
		DamageMultiplierAdditive: 1 + 0.02*float64(priest.Talents.SpiritualHealing),
		ThreatMultiplier:         1,
	}
}

// Heal and Greater Heal get Improved Healing and Divine Fury; Flash Heal gets neither.
func (priest *Priest) registerDirectHeal(ranks []HealRankData, spellCode int32, isSlowHeal bool) []*core.Spell {
	spells := make([]*core.Spell, len(ranks)+1)

	for i, rank := range ranks {
		if rank.Level > int32(priest.Level) {
			continue
		}

		config := priest.healSpellConfig(rank, i+1, spellCode)
		config.BonusCoefficient = rank.directCoefficient()

		if isSlowHeal {
			config.ManaCost.Multiplier = 100 - 5*priest.Talents.ImprovedHealing
			config.Cast.DefaultCast.CastTime -= time.Millisecond * 100 * time.Duration(priest.Talents.DivineFury)
		}

		minHeal := rank.Min + rank.levelBonus(int32(priest.Level))
		maxHeal := rank.Max + rank.levelBonus(int32(priest.Level))
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.CalcAndDealHealing(sim, target, sim.Roll(minHeal, maxHeal), spell.OutcomeHealingCrit)
		}

		spells[i+1] = priest.GetOrRegisterSpell(config)
	}

	return spells
}

func (priest *Priest) registerRenew() {
	priest.Renew = make([]*core.Spell, len(RenewRanks)+1)

	for i, rank := range RenewRanks {
		if rank.Level > int32(priest.Level) {
			continue
		}

		config := priest.healSpellConfig(rank, i+1, SpellCode_PriestRenew)
		// Improved Renew: +5% per rank, additive with Spiritual Healing.
		config.DamageMultiplierAdditive += 0.05 * float64(priest.Talents.ImprovedRenew)

		tickHeal := rank.Min
		tickCoefficient := RenewDuration.Seconds() / 15 / RenewTicks * rank.levelPenalty()
		config.Hot = core.DotConfig{
			Aura: core.Aura{
				Label: "Renew",
			},
			NumberOfTicks: RenewTicks,
			TickLength:    RenewDuration / RenewTicks,
			OnSnapshot: func(sim *core.Simulation, target *core.Unit, dot *core.Dot, _ bool) {
				dot.SnapshotBaseDamage = tickHeal + tickCoefficient*dot.Spell.HealingPower(target)
				dot.SnapshotAttackerMultiplier = dot.Spell.CasterHealingMultiplier()
			},
			OnTick: func(sim *core.Simulation, target *core.Unit, dot *core.Dot) {
				dot.CalcAndDealPeriodicSnapshotHealing(sim, target, dot.OutcomeTick)
			},
		}
		config.ApplyEffects = func(sim *core.Simulation, target *core.Unit, spell *core.Spell) {
			spell.Hot(target).Apply(sim)
		}

		priest.Renew[i+1] = priest.GetOrRegisterSpell(config)
	}
}

// Prayer of Healing heals every player in the priest's party, each with its own crit roll.
func (priest *Priest) registerPrayerOfHealing() {
	priest.PrayerOfHealing = make([]*core.Spell, len(PrayerOfHealingRanks)+1)

	for i, rank := range PrayerOfHealingRanks {
		if rank.Level > int32(priest.Level) {
			continue
		}

		config := priest.healSpellConfig(rank, i+1, SpellCode_PriestPrayerOfHealing)
		config.BonusCoefficient = PrayerOfHealingCoefficient * rank.levelPenalty()
		config.ManaCost.Multiplier = 100 - 10*priest.Talents.ImprovedPrayerOfHealing

		minHeal := rank.Min + rank.levelBonus(int32(priest.Level))
		maxHeal := rank.Max + rank.levelBonus(int32(priest.Level))
		config.ApplyEffects = func(sim *core.Simulation, _ *core.Unit, spell *core.Spell) {
			for _, agent := range priest.Party.Players {
				target := &agent.GetCharacter().Unit
				spell.CalcAndDealHealing(sim, target, sim.Roll(minHeal, maxHeal), spell.OutcomeHealingCrit)
			}
		}

		priest.PrayerOfHealing[i+1] = priest.GetOrRegisterSpell(config)
	}
}
