package warriorvalidation

import (
	"math"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/ruleset"
	"google.golang.org/protobuf/encoding/protojson"
)

const protTalents = "20304300302-03-55200110530201051"

// Boss used by the tank checks: a level 63 raid boss hitting for 1500-2000 every 2 seconds,
// plus 805 attack power (805 / 14 x 2 = 115 more per hit).
var tankBoss = &proto.Target{
	Level:         63,
	MobType:       proto.MobType_MobTypeUnknown,
	Stats:         stats.Stats{stats.Armor: 3731, stats.AttackPower: 805}.ToFloatArray(),
	MinBaseDamage: 1500,
	DamageSpread:  1.0 / 3,
	SwingSpeed:    2,
	TankIndex:     0,
}

func loadGearSet(t *testing.T, path string) *proto.EquipmentSpec {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	gear := &proto.EquipmentSpec{}
	if err := protojson.Unmarshal(data, gear); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return gear
}

func tankRequest(t *testing.T, apl string, iterations int32) *proto.RaidSimRequest {
	if err := ruleset.Activate("vanilla"); err != nil {
		t.Fatal(err)
	}
	player := core.WithSpec(&proto.Player{
		Name:               "Tank",
		Race:               proto.Race_RaceHuman,
		Class:              proto.Class_ClassWarrior,
		Equipment:          loadGearSet(t, "../../ui/tank_warrior/gear_sets/p0.bis.gear.json"),
		TalentsString:      protTalents,
		Consumes:           &proto.Consumes{},
		Buffs:              &proto.IndividualBuffs{},
		Rotation:           mustAPL(t, apl),
		DistanceFromTarget: 5,
		InFrontOfTarget:    true,
	}, &proto.Player_TankWarrior{TankWarrior: &proto.TankWarrior{Options: &proto.TankWarrior_Options{
		Stance: proto.WarriorStance_WarriorStanceDefensive,
	}}})

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	raid.Tanks = []*proto.UnitReference{{Type: proto.UnitReference_Player, Index: 0}}
	return &proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: 120,
			Targets:  []*proto.Target{tankBoss},
		},
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 101},
	}
}

func protAPL(t *testing.T) string {
	t.Helper()
	apl, err := os.ReadFile("../../ui/tank_warrior/apls/prot.apl.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(apl)
}

func runTank(t *testing.T, req *proto.RaidSimRequest) (*proto.UnitMetrics, *proto.UnitMetrics) {
	t.Helper()
	res := core.RunRaidSim(req)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	return res.RaidMetrics.Parties[0].Players[0], res.EncounterMetrics.Targets[0]
}

// Metrics of one action against the boss (target index 0).
func onBoss(unit *proto.UnitMetrics, id *proto.ActionID) *proto.TargetedActionMetrics {
	return actionTotals(unit, id, true)
}

// Metrics of one of the boss's actions, summed over everyone it hit.
func byBoss(boss *proto.UnitMetrics, id *proto.ActionID) *proto.TargetedActionMetrics {
	return actionTotals(boss, id, false)
}

func actionTotals(unit *proto.UnitMetrics, id *proto.ActionID, onlyFirstTarget bool) *proto.TargetedActionMetrics {
	total := &proto.TargetedActionMetrics{}
	for _, a := range unit.Actions {
		if !protoEqualID(a.Id, id) {
			continue
		}
		for _, tg := range a.Targets {
			if onlyFirstTarget && tg.UnitIndex != 0 {
				continue
			}
			total.Casts += tg.Casts
			total.Hits += tg.Hits
			total.Crits += tg.Crits
			total.Crushes += tg.Crushes
			total.Misses += tg.Misses
			total.Dodges += tg.Dodges
			total.Parries += tg.Parries
			total.Blocks += tg.Blocks
			total.BlockedCrits += tg.BlockedCrits
			total.Glances += tg.Glances
			total.Damage += tg.Damage
			total.CritDamage += tg.CritDamage
			total.CrushDamage += tg.CrushDamage
			total.BlockDamage += tg.BlockDamage
			total.Threat += tg.Threat
		}
	}
	return total
}

func protoEqualID(a, b *proto.ActionID) bool {
	return a.GetSpellId() == b.GetSpellId() && a.GetOtherId() == b.GetOtherId() && a.GetItemId() == b.GetItemId() && a.Tag == b.Tag
}

func spellID(id int32) *proto.ActionID { return &proto.ActionID{RawId: &proto.ActionID_SpellId{SpellId: id}} }

// Defensive Stance (1.3x) with 5/5 Defiance (1.15x).
// Source: https://github.com/magey/classic-warrior/wiki/Threat-Mechanics#threat-multipliers
const protThreatMultiplier = 1.3 * 1.15

// Checks each ability's threat against the values measured in game by the classic-warrior
// project (patch 1.13.6): threat = (damage x ability multiplier + bonus threat per landed hit) x stance.
func TestTankThreat(t *testing.T) {
	player, _ := runTank(t, tankRequest(t, protAPL(t), 2000))

	type threatRule struct {
		name       string
		id         *proto.ActionID
		multiplier float64 // on damage
		bonus      float64 // per landed hit
	}
	rules := []threatRule{
		{"Auto attacks", &proto.ActionID{RawId: &proto.ActionID_OtherId{OtherId: proto.OtherAction_OtherActionAttack}, Tag: 1}, 1, 0},
		{"Revenge (rank 6)", spellID(25288), 2.25, 270},
		{"Shield Slam (rank 4)", spellID(23925), 1, 254},
		{"Sunder Armor (rank 5)", spellID(11597), 1, 261},
		{"Deep Wounds", spellID(12867), 1, 0},
	}
	for _, r := range rules {
		m := onBoss(player, r.id)
		if m.Casts == 0 && m.Damage == 0 {
			t.Errorf("%s: never used", r.name)
			continue
		}
		landed := float64(m.Hits + m.Crits + m.Blocks + m.BlockedCrits + m.Glances)
		want := (m.Damage*r.multiplier + r.bonus*landed) * protThreatMultiplier
		t.Logf("%-22s threat %10.0f, expected %10.0f", r.name, m.Threat, want)
		if math.Abs(m.Threat-want) > 1e-6*want+1 {
			t.Errorf("%s: threat %.0f, expected %.0f", r.name, m.Threat, want)
		}
	}

	// Rage from Bloodrage and Shield Specialization: 5 threat per rage, not changed by stance.
	// Rage from dealing and taking damage gives no threat. Same source as above.
	rageFromEffects := 0.0
	refunds := 0.0
	for _, r := range player.Resources {
		if r.Type != proto.ResourceType_ResourceTypeRage {
			continue
		}
		switch r.Id.GetSpellId() {
		case 2687, 12727: // Bloodrage, Shield Specialization
			rageFromEffects += r.ActualGain
		}
		if r.Id.GetOtherId() == proto.OtherAction_OtherActionRefund {
			refunds = r.ActualGain
		}
	}
	rage := onBoss(player, &proto.ActionID{RawId: &proto.ActionID_OtherId{OtherId: proto.OtherAction_OtherActionRageGain}})
	t.Logf("Rage gain threat %.0f, expected %.0f", rage.Threat, 5*rageFromEffects)
	if math.Abs(rage.Threat-5*rageFromEffects) > 1 {
		t.Errorf("rage gain threat %.0f, expected %.0f", rage.Threat, 5*rageFromEffects)
	}

	// Abilities that miss, are dodged or are parried refund 80% of their rage cost
	// (WarriorSim does the same).
	failed := 0.0
	for id, cost := range map[int32]float64{25288: 5, 23925: 20, 11597: 12, 25286: 12} {
		m := onBoss(player, spellID(id))
		failed += float64(m.Misses+m.Dodges+m.Parries) * 0.8 * cost
	}
	t.Logf("Rage refunded %.0f, expected %.0f", refunds, failed)
	if math.Abs(refunds-failed) > 1e-3*failed {
		t.Errorf("rage refunded %.0f, expected %.0f", refunds, failed)
	}
}

// Expected share of the boss's swings for each outcome, from the warrior's character sheet.
// Vanilla attack table for a mob hitting a player, one roll in this order: miss, dodge, parry,
// block, crit, crushing blow, hit. Each point of the mob's weapon skill above the player's
// defense moves 0.04% from miss/dodge/parry/block to crit; a level 63 boss has 315 skill.
// Crushing blows: (skill - 5 x player level) x 2% - 15% = 15% for a level 63 boss.
// Source: https://github.com/magey/classic-warrior/wiki/Attack-table and
// https://wowwiki-archive.fandom.com/wiki/Crushing_blow
func expectedBossTable(sheet stats.Stats) map[string]float64 {
	const skillGap = 315 - 300.0
	defense := 300 + sheet[stats.Defense]
	table := map[string]float64{
		"miss":  0.05 + (defense-315)*0.0004,
		"dodge": sheet[stats.Dodge]/100 - skillGap*0.0004, // the sheet already includes bonus defense
		"parry": sheet[stats.Parry]/100 - skillGap*0.0004,
		"block": sheet[stats.Block]/100 - skillGap*0.0004,
		"crit":  0.05 + (315-defense)*0.0004,
		"crush": skillGap*0.02 - 0.15,
	}
	sum := 0.0
	for _, v := range table {
		sum += v
	}
	table["hit"] = 1 - sum
	return table
}

func sheetStats(t *testing.T, req *proto.RaidSimRequest) stats.Stats {
	t.Helper()
	cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: req.Raid, Encounter: req.Encounter})
	if cs.ErrorResult != "" {
		t.Fatal(cs.ErrorResult)
	}
	return stats.FromFloatArray(cs.RaidStats.Parties[0].Players[0].FinalStats.Stats)
}

// A warrior standing in Defensive Stance and only auto-attacking, so nothing changes its
// avoidance during the fight (Shield Block would).
const whiteOnlyAPL = `{"type":"TypeAPL","priorityList":[]}`

func TestTankBossAttackTable(t *testing.T) {
	req := tankRequest(t, whiteOnlyAPL, 2000)
	sheet := sheetStats(t, req)
	_, boss := runTank(t, req)
	m := byBoss(boss, &proto.ActionID{RawId: &proto.ActionID_OtherId{OtherId: proto.OtherAction_OtherActionAttack}, Tag: 1})
	n := float64(m.Casts)
	if n == 0 {
		t.Fatal("the boss never swung")
	}
	got := map[string]float64{
		"miss": float64(m.Misses) / n, "dodge": float64(m.Dodges) / n, "parry": float64(m.Parries) / n,
		"block": float64(m.Blocks+m.BlockedCrits) / n, "crit": float64(m.Crits) / n,
		"crush": float64(m.Crushes) / n, "hit": float64(m.Hits) / n,
	}
	want := expectedBossTable(sheet)
	t.Logf("%d boss swings; defense %.0f, dodge %.2f%%, parry %.2f%%, block %.2f%%", m.Casts, 300+sheet[stats.Defense], sheet[stats.Dodge], sheet[stats.Parry], sheet[stats.Block])
	for _, k := range []string{"miss", "dodge", "parry", "block", "crit", "crush", "hit"} {
		se := math.Sqrt(want[k] * (1 - want[k]) / n)
		t.Logf("%-6s %6.2f%%  expected %6.2f%% ± %.2f%%", k, 100*got[k], 100*want[k], 300*se)
		if math.Abs(got[k]-want[k]) > 3*se {
			t.Errorf("%s: %.2f%%, expected %.2f%% ± %.2f%%", k, 100*got[k], 100*want[k], 300*se)
		}
	}
}

// Damage the boss deals, worked out by hand from its swing damage and the warrior's armour,
// for the number of hits, crits, crushing blows and blocks the sim rolled.
// - Armour: reduction = armour / (armour + 400 + 85 x attacker level), capped at 75%.
//   Source: https://wowwiki-archive.fandom.com/wiki/Armor (vanilla formula).
// - Defensive Stance: 10% less damage taken.
// - Attack power adds AP / 14 x swing time to each hit, as for players.
// - Crits deal 200%, crushing blows 150%; a block subtracts the shield's block value.
func TestTankDamageTaken(t *testing.T) {
	req := tankRequest(t, whiteOnlyAPL, 2000)
	sheet := sheetStats(t, req)
	player, boss := runTank(t, req)
	m := byBoss(boss, &proto.ActionID{RawId: &proto.ActionID_OtherId{OtherId: proto.OtherAction_OtherActionAttack}, Tag: 1})

	if m.Casts == 0 {
		t.Fatal("the boss never swung")
	}
	apBonus := 805.0 / 14 * tankBoss.SwingSpeed
	low := tankBoss.MinBaseDamage + apBonus
	high := tankBoss.MinBaseDamage*(1+tankBoss.DamageSpread) + apBonus
	armor := sheet[stats.Armor]
	scale := (1 - min(armor/(armor+400+85*63), 0.75)) * 0.9
	avg := (low + high) / 2 * scale

	// Blocked hits: average of max(0, x - block value) for x uniform on [low, high] x scale.
	bv := blockValue(sheet)
	a, b := low*scale, high*scale
	blocked := 0.0
	if bv < b {
		lo := max(a, bv)
		blocked = ((b-bv)*(b-bv) - (lo-bv)*(lo-bv)) / 2 / (b - a)
	}

	want := float64(m.Hits)*avg + float64(m.Crits)*2*avg + float64(m.Crushes)*1.5*avg + float64(m.Blocks)*blocked
	t.Logf("armour %.0f, block value %.0f: expected %.0f damage, sim %.0f (%+.2f%%); DTPS %.1f", armor, bv, want, m.Damage, 100*(m.Damage/want-1), player.Dtps.Avg)
	if m.BlockedCrits > 0 {
		t.Errorf("boss swings can't be blocked crits in vanilla, got %d", m.BlockedCrits)
	}
	if math.Abs(m.Damage/want-1) > 0.005 {
		t.Errorf("damage taken %.0f, expected %.0f", m.Damage, want)
	}
}

// Shield block value: the shield's own value plus 1 per 20 strength, minus 1.
// Source: https://wowwiki-archive.fandom.com/wiki/Block_value
func blockValue(sheet stats.Stats) float64 {
	return sheet[stats.BlockValue] + max(0, sheet[stats.Strength]/20-1)
}
