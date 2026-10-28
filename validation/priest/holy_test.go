package priestvalidation

import (
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/priest"
	"github.com/wowsims/classic/sim/ruleset"
	_ "github.com/wowsims/classic/sim/ruleset/vanilla"
	"google.golang.org/protobuf/encoding/protojson"
)

// 20/31/0: Mental Agility, Meditation 3/3, Inner Focus, Holy Specialization 4/5,
// Divine Fury 5/5, Improved Healing 3/3, Spiritual Guidance 5/5, Spiritual Healing 5/5.
const holyTalents = "05420013050-034050032300155"

const (
	holySpec          = 4 // % crit on Holy spells
	spiritualHealing  = 5 // +2% healing per point
	improvedHealing   = 3 // -5% mana on Heal and Greater Heal per point
	improvedRenew     = 3 // +5% Renew healing per point
	meditation        = 3 // 5% of spirit regen while casting per point
	divineFury        = 5 // -0.1s on Heal and Greater Heal per point
	mentalAgility     = 5 // -2% mana on instant spells per point
	spiritualGuidance = 5 // 5% of spirit as +healing per point
)

// A boss that never attacks, so nothing but the priest's own casts changes the result.
var dummyBoss = &proto.Target{
	Level:   63,
	MobType: proto.MobType_MobTypeUnknown,
	Stats:   stats.Stats{stats.Armor: 3731}.ToFloatArray(),
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

func mustAPL(t *testing.T, js string) *proto.APLRotation {
	t.Helper()
	apl := &proto.APLRotation{}
	if err := protojson.Unmarshal([]byte(js), apl); err != nil {
		t.Fatalf("bad APL: %v", err)
	}
	return apl
}

func castOnly(spellID int32) string {
	return fmt.Sprintf(`{"type":"TypeAPL","priorityList":[{"action":{"castSpell":{"spellId":{"spellId":%d}}}}]}`, spellID)
}

func holyRequest(t *testing.T, apl string, duration float64, iterations int32) *proto.RaidSimRequest {
	t.Helper()
	if err := ruleset.Activate("vanilla"); err != nil {
		t.Fatal(err)
	}
	player := core.WithSpec(&proto.Player{
		Name:          "Priest",
		Race:          proto.Race_RaceUndead,
		Class:         proto.Class_ClassPriest,
		Equipment:     loadGearSet(t, "../../ui/healing_priest/gear_sets/p0.holy.gear.json"),
		TalentsString: holyTalents,
		Consumes:      &proto.Consumes{},
		Buffs:         &proto.IndividualBuffs{},
		Rotation:      mustAPL(t, apl),
	}, &proto.Player_HealingPriest{HealingPriest: &proto.HealingPriest{Options: &proto.HealingPriest_Options{}}})

	raid := core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, &proto.RaidBuffs{}, &proto.Debuffs{})
	return &proto.RaidSimRequest{
		Raid: raid,
		Encounter: &proto.Encounter{
			Duration: duration,
			Targets:  []*proto.Target{dummyBoss},
		},
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 7},
	}
}

func run(t *testing.T, req *proto.RaidSimRequest) *proto.UnitMetrics {
	t.Helper()
	res := core.RunRaidSim(req)
	if res.Error != nil {
		t.Fatal(res.Error)
	}
	return res.RaidMetrics.Parties[0].Players[0]
}

func sheetStats(t *testing.T, req *proto.RaidSimRequest) stats.Stats {
	t.Helper()
	cs := core.ComputeStats(&proto.ComputeStatsRequest{Raid: req.Raid, Encounter: req.Encounter})
	if cs.ErrorResult != "" {
		t.Fatal(cs.ErrorResult)
	}
	return stats.FromFloatArray(cs.RaidStats.Parties[0].Players[0].FinalStats.Stats)
}

func isSpell(id *proto.ActionID, spellID int32) bool {
	return id.GetSpellId() == spellID
}

// Healing metrics of one spell, summed over every target it healed.
func healTotals(unit *proto.UnitMetrics, spellID int32) *proto.TargetedActionMetrics {
	total := &proto.TargetedActionMetrics{}
	for _, a := range unit.Actions {
		if !isSpell(a.Id, spellID) {
			continue
		}
		for _, tg := range a.Targets {
			total.Casts += tg.Casts
			total.Hits += tg.Hits
			total.Crits += tg.Crits
			total.Ticks += tg.Ticks
			total.CritTicks += tg.CritTicks
			total.Healing += tg.Healing
			total.CritHealing += tg.CritHealing
		}
	}
	return total
}

// Mana spent on one spell, from the resource metrics.
func manaSpent(unit *proto.UnitMetrics, spellID int32) (casts int32, mana float64) {
	for _, r := range unit.Resources {
		if r.Type == proto.ResourceType_ResourceTypeMana && isSpell(r.Id, spellID) {
			casts += r.Events
			mana -= r.Gain
		}
	}
	return
}

func find(ranks []priest.HealRankData, spellID int32) priest.HealRankData {
	for _, r := range ranks {
		if r.SpellID == spellID {
			return r
		}
	}
	panic(fmt.Sprintf("no rank %d", spellID))
}

func within(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	diff := (got - want) / want
	status := "ok"
	if math.Abs(diff) > tolerance {
		status = "FAIL"
		t.Errorf("%s: got %.2f, hand calculation %.2f (%+.2f%%, allowed %.2f%%)", name, got, want, diff*100, tolerance*100)
	}
	t.Logf("%-38s sim %10.2f  hand %10.2f  %+6.2f%%  %s", name, got, want, diff*100, status)
}

// Direct heals: average non-crit size, crit size and crit rate against
//
//	(rank average + level scaling + coefficient x +healing) x (1 + Spiritual Healing)
//
// with crits at 150%.
func TestHolyDirectHeals(t *testing.T) {
	cases := []struct {
		name      string
		spellID   int32
		ranks     []priest.HealRankData
		coeff     float64
		manaMult  float64
		castTime  float64
		perTarget float64
	}{
		{"Greater Heal rank 5", 25314, priest.GreaterHealRanks, 3.0 / 3.5, 1 - 0.05*improvedHealing, 3 - 0.1*divineFury, 1},
		{"Greater Heal rank 1", 2060, priest.GreaterHealRanks, 3.0 / 3.5, 1 - 0.05*improvedHealing, 3 - 0.1*divineFury, 1},
		{"Heal rank 4", 6064, priest.HealRanks, 3.0 / 3.5, 1 - 0.05*improvedHealing, 3 - 0.1*divineFury, 1},
		{"Heal rank 1 (below level 20)", 2054, priest.HealRanks, 3.0 / 3.5 * (1 - 4*0.0375), 1 - 0.05*improvedHealing, 3 - 0.1*divineFury, 1},
		{"Flash Heal rank 7", 10917, priest.FlashHealRanks, 1.5 / 3.5, 1, 1.5, 1},
		{"Prayer of Healing rank 5", 25316, priest.PrayerOfHealingRanks, 3.0 / 3.5 / 3, 1, 3, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Fights short enough that no spell runs the priest out of mana.
			const fightLength = 14.9
			req := holyRequest(t, castOnly(c.spellID), fightLength, 3000)
			sheet := sheetStats(t, req)
			unit := run(t, req)
			m := healTotals(unit, c.spellID)
			if m.Hits+m.Crits == 0 {
				t.Fatal("the spell never landed")
			}

			rank := find(c.ranks, c.spellID)
			levels := math.Max(0, float64(min(60, rank.MaxLevel)-rank.Level))
			healingPower := sheet[stats.SpellPower] + sheet[stats.HealingPower]
			want := ((rank.Min+rank.Max)/2 + levels*rank.PerLevel + c.coeff*healingPower) * (1 + 0.02*spiritualHealing)

			nonCrit := (m.Healing - m.CritHealing) / float64(m.Hits)
			crit := m.CritHealing / float64(m.Crits)
			within(t, "average non-crit heal", nonCrit, want, 0.005)
			within(t, "average crit heal", crit, want*1.5, 0.01)

			n := float64(m.Hits + m.Crits)
			p := sheet[stats.SpellCrit]/100 + holySpec/100.0
			gotCrit := float64(m.Crits) / n
			se := math.Sqrt(p * (1 - p) / n)
			if math.Abs(gotCrit-p) > 3*se {
				t.Errorf("crit rate %.2f%%, expected %.2f%% (+/- %.2f)", gotCrit*100, p*100, 3*se*100)
			}
			t.Logf("crit rate %.2f%% (expected %.2f%%)", gotCrit*100, p*100)

			casts, mana := manaSpent(unit, c.spellID)
			if casts == 0 {
				t.Fatal("no mana spent")
			}
			within(t, "mana per cast", mana/float64(casts), rank.Mana*c.manaMult, 0.0001)

			// Cast time: in a 14.9s fight the priest finishes floor(14.9 / cast time) casts.
			var completed int32
			for _, a := range unit.Actions {
				if isSpell(a.Id, c.spellID) {
					for _, tg := range a.Targets {
						completed = max(completed, tg.Casts)
					}
				}
			}
			within(t, "casts per fight", float64(completed)/float64(req.SimOptions.Iterations), math.Floor(fightLength/c.castTime), 0.0001)
		})
	}
}

// Renew: 5 ticks, each (rank tick + 20% of +healing) x (1 + Spiritual Healing + Improved Renew), no crits.
func TestHolyRenew(t *testing.T) {
	const spellID = 25315
	apl := fmt.Sprintf(`{"type":"TypeAPL","priorityList":[{"action":{"condition":{"not":{"val":{"dotIsActive":{"spellId":{"spellId":%d}}}}},"castSpell":{"spellId":{"spellId":%d}}}}]}`, spellID, spellID)
	req := holyRequest(t, apl, 60, 100)
	sheet := sheetStats(t, req)
	unit := run(t, req)
	m := healTotals(unit, spellID)
	if m.Ticks == 0 {
		t.Fatal("Renew never ticked")
	}
	if m.CritTicks != 0 {
		t.Errorf("Renew ticks crit %d times; they can't crit in 1.12", m.CritTicks)
	}

	rank := find(priest.RenewRanks, spellID)
	healingPower := sheet[stats.SpellPower] + sheet[stats.HealingPower]
	want := (rank.Min + 0.2*healingPower) * (1 + 0.02*spiritualHealing + 0.05*improvedRenew)
	within(t, "Renew tick", m.Healing/float64(m.Ticks), want, 0.0001)

	casts, mana := manaSpent(unit, spellID)
	within(t, "Renew mana per cast", mana/float64(casts), rank.Mana*(1-0.02*mentalAgility), 0.0001)
}

// Mana and regen: max mana, spirit regen while casting (Meditation) and outside the five-second rule.
func TestHolyManaAndRegen(t *testing.T) {
	// Cast Greater Heal rank 1 for the first 30s, then stop, so the fight has both kinds of regen.
	apl := `{"type":"TypeAPL","priorityList":[{"action":{"condition":{"cmp":{"op":"OpLt","lhs":{"currentTime":{}},"rhs":{"const":{"val":"30s"}}}},"castSpell":{"spellId":{"spellId":2060}}}}]}`
	req := holyRequest(t, apl, 90, 20)
	sheet := sheetStats(t, req)
	unit := run(t, req)

	// Level 60 priest base mana as the engine has it (sim/core/base_stats.go), 15 mana per
	// Intellect after the first 20. UNCERTAIN: the 1.12 server table (cmangos classic-db
	// player_classlevelstats) says 1,436, 60 more than the engine; mage is off by the same 60.
	baseMana := 1376.0
	intel := sheet[stats.Intellect]
	wantMana := baseMana + 20 + (intel-20)*15
	within(t, "max mana", sheet[stats.Mana], wantMana, 0.0001)

	// 1.12 priest regen: 12.5 + Spirit/4 every 2s (vmangos Player::OCTRegenMPPerSpirit).
	spiritTick := 12.5 + sheet[stats.Spirit]/4
	mp5Tick := sheet[stats.MP5] / 5 * 2
	for _, r := range unit.Resources {
		if r.Type != proto.ResourceType_ResourceTypeMana || r.Id.GetOtherId() != proto.OtherAction_OtherActionManaRegen || r.Events == 0 {
			continue
		}
		perTick := r.Gain / float64(r.Events)
		switch r.Id.Tag {
		case 1:
			within(t, "regen tick while casting", perTick, mp5Tick+spiritTick*0.05*meditation, 0.0001)
		case 2:
			within(t, "regen tick outside 5s rule", perTick, mp5Tick+spiritTick, 0.0001)
		}
	}
}

// Mental Strength adds 2% max mana per point; it does not touch Intellect.
func TestMentalStrength(t *testing.T) {
	req := holyRequest(t, castOnly(2060), 10, 1)
	without := sheetStats(t, req)
	req.Raid.Parties[0].Players[0].TalentsString = "054200130505-034050032300155"
	with := sheetStats(t, req)
	within(t, "Intellect unchanged", with[stats.Intellect], without[stats.Intellect], 0.0001)
	within(t, "max mana with 5/5 Mental Strength", with[stats.Mana], without[stats.Mana]*1.10, 0.0001)
}
