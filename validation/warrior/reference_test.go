// Package warriorvalidation compares the warrior sim with WarriorSim
// (https://github.com/GuybrushGit/WarriorSim), an independent Classic warrior
// simulator, on the shared scenarios in scenarios.json.
//
// Reference numbers are recorded in reference_results.json by
// run_warriorsim.mjs; see README.md for how to refresh them.
package warriorvalidation

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
	"github.com/wowsims/classic/sim/ruleset"
	_ "github.com/wowsims/classic/sim/ruleset/vanilla"
	"google.golang.org/protobuf/encoding/protojson"
)

type scenario struct {
	Name           string           `json:"name"`
	Race           string           `json:"race"`
	Talents        string           `json:"talents"`
	Gear           map[string]int32 `json:"gear"`
	Buffs          []string         `json:"buffs"`
	Rotation       string           `json:"rotation"`
	FightSeconds   float64          `json:"fightSeconds"`
	ExecutePercent float64          `json:"executePercent"`
	TargetArmor    float64          `json:"targetArmor"`
	// Tolerance is the allowed relative DPS difference from the reference.
	Tolerance float64 `json:"tolerance"`
}

type weaponResult struct {
	TotalDamage float64            `json:"totalDamage"`
	Outcomes    map[string]float64 `json:"outcomes"`
}

type referenceResult struct {
	Iterations int           `json:"iterations"`
	DPS        float64       `json:"dps"`
	DPSStderr  float64       `json:"dpsStderr"`
	MH         *weaponResult `json:"mh"`
	OH         *weaponResult `json:"oh"`
	Abilities  map[string]struct {
		DPS   float64 `json:"dps"`
		Casts float64 `json:"casts"`
	} `json:"abilities"`
}

var races = map[string]proto.Race{
	"Human": proto.Race_RaceHuman, "Orc": proto.Race_RaceOrc, "Dwarf": proto.Race_RaceDwarf,
	"Night Elf": proto.Race_RaceNightElf, "Undead": proto.Race_RaceUndead, "Tauren": proto.Race_RaceTauren,
	"Gnome": proto.Race_RaceGnome, "Troll": proto.Race_RaceTroll,
}

var slots = map[string]proto.ItemSlot{
	"head": proto.ItemSlot_ItemSlotHead, "neck": proto.ItemSlot_ItemSlotNeck, "shoulder": proto.ItemSlot_ItemSlotShoulder,
	"back": proto.ItemSlot_ItemSlotBack, "chest": proto.ItemSlot_ItemSlotChest, "wrist": proto.ItemSlot_ItemSlotWrist,
	"hands": proto.ItemSlot_ItemSlotHands, "waist": proto.ItemSlot_ItemSlotWaist, "legs": proto.ItemSlot_ItemSlotLegs,
	"feet": proto.ItemSlot_ItemSlotFeet, "finger1": proto.ItemSlot_ItemSlotFinger1, "finger2": proto.ItemSlot_ItemSlotFinger2,
	"trinket1": proto.ItemSlot_ItemSlotTrinket1, "trinket2": proto.ItemSlot_ItemSlotTrinket2,
	"mainhand": proto.ItemSlot_ItemSlotMainHand, "offhand": proto.ItemSlot_ItemSlotOffHand, "ranged": proto.ItemSlot_ItemSlotRanged,
}

func loadJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
}

func mustAPL(t *testing.T, js string) *proto.APLRotation {
	t.Helper()
	apl := &proto.APLRotation{}
	if err := protojson.Unmarshal([]byte(js), apl); err != nil {
		t.Fatalf("bad APL: %v", err)
	}
	return apl
}

func buildRequest(t *testing.T, sc scenario, iterations int32) *proto.RaidSimRequest {
	race, ok := races[sc.Race]
	if !ok {
		t.Fatalf("unknown race %q", sc.Race)
	}
	equipment := &proto.EquipmentSpec{Items: make([]*proto.ItemSpec, 17)}
	for i := range equipment.Items {
		equipment.Items[i] = &proto.ItemSpec{}
	}
	for slot, id := range sc.Gear {
		s, ok := slots[slot]
		if !ok {
			t.Fatalf("unknown slot %q", slot)
		}
		equipment.Items[s] = &proto.ItemSpec{Id: id}
	}

	stance := proto.WarriorStance_WarriorStanceBattle
	individualBuffs := &proto.IndividualBuffs{}
	raidBuffs := &proto.RaidBuffs{}
	consumes := &proto.Consumes{}
	for _, b := range sc.Buffs {
		switch b {
		case "kings":
			individualBuffs.BlessingOfKings = true
		case "might_improved":
			individualBuffs.BlessingOfMight = proto.TristateEffect_TristateEffectImproved
		case "motw_improved":
			raidBuffs.GiftOfTheWild = proto.TristateEffect_TristateEffectImproved
		case "leader_of_the_pack":
			raidBuffs.LeaderOfThePack = true
		case "mongoose":
			consumes.AgilityElixir = proto.AgilityElixir_ElixirOfTheMongoose
		case "juju_power":
			consumes.StrengthBuff = proto.StrengthBuff_JujuPower
		case "juju_might":
			consumes.AttackPowerBuff = proto.AttackPowerBuff_JujuMight
		case "dumplings":
			consumes.Food = proto.Food_FoodSmokedDesertDumpling
		case "battle_stance":
			stance = proto.WarriorStance_WarriorStanceBattle
		case "berserker_stance":
			stance = proto.WarriorStance_WarriorStanceBerserker
		default:
			t.Fatalf("buff %q not mapped yet", b)
		}
	}

	var rotation *proto.APLRotation
	switch sc.Rotation {
	case "white":
		// An empty priority list: auto attacks only.
		rotation = &proto.APLRotation{Type: proto.APLRotation_TypeAPL}
	case "fury":
		// Must match ROTATIONS.fury in run_warriorsim.mjs.
		rotation = mustAPL(t, `{"type":"TypeAPL","priorityList":[
			{"action":{"castSpell":{"spellId":{"spellId":2687}}}},
			{"action":{"castSpell":{"spellId":{"spellId":23894}}}},
			{"action":{"castSpell":{"spellId":{"spellId":1680}}}},
			{"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentRage":{}},"rhs":{"const":{"val":"30"}}}},"castSpell":{"spellId":{"spellId":25286,"tag":1}}}}
		]}`)
	case "arms":
		// Must match ROTATIONS.arms in run_warriorsim.mjs.
		rotation = mustAPL(t, `{"type":"TypeAPL","priorityList":[
			{"action":{"castSpell":{"spellId":{"spellId":2687}}}},
			{"action":{"castSpell":{"spellId":{"spellId":21553}}}},
			{"action":{"castSpell":{"spellId":{"spellId":1680}}}},
			{"action":{"condition":{"cmp":{"op":"OpGe","lhs":{"currentRage":{}},"rhs":{"const":{"val":"50"}}}},"castSpell":{"spellId":{"spellId":25286,"tag":1}}}}
		]}`)
	default:
		t.Fatalf("rotation %q not mapped yet", sc.Rotation)
	}

	player := core.WithSpec(&proto.Player{
		Name:               sc.Name,
		Race:               race,
		Class:              proto.Class_ClassWarrior,
		Equipment:          equipment,
		TalentsString:      sc.Talents,
		Consumes:           consumes,
		Buffs:              individualBuffs,
		Rotation:           rotation,
		DistanceFromTarget: 5,
	}, &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{Stance: stance}}})

	return &proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(player, &proto.PartyBuffs{}, raidBuffs, &proto.Debuffs{}),
		Encounter: &proto.Encounter{
			Duration:             sc.FightSeconds,
			ExecuteProportion_20: sc.ExecutePercent / 100,
			ExecuteProportion_25: sc.ExecutePercent / 100,
			ExecuteProportion_35: sc.ExecutePercent / 100,
			Targets: []*proto.Target{{
				Level:   63,
				MobType: proto.MobType_MobTypeUnknown,
				Stats:   stats.Stats{stats.Armor: sc.TargetArmor}.ToFloatArray(),
			}},
		},
		SimOptions: &proto.SimOptions{Iterations: iterations, RandomSeed: 101},
	}
}

func TestWarriorAgainstWarriorSim(t *testing.T) {
	if err := ruleset.Activate("vanilla"); err != nil {
		t.Fatal(err)
	}
	var file struct {
		Scenarios []scenario `json:"scenarios"`
	}
	loadJSON(t, "scenarios.json", &file)
	var ref struct {
		WarriorSimCommit string                     `json:"warriorSimCommit"`
		Scenarios        map[string]referenceResult `json:"scenarios"`
	}
	loadJSON(t, "reference_results.json", &ref)

	const iterations = 5000
	for _, sc := range file.Scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			want, ok := ref.Scenarios[sc.Name]
			if !ok {
				t.Fatalf("no reference result for %s; run run_warriorsim.mjs", sc.Name)
			}
			result := core.RunRaidSim(buildRequest(t, sc, iterations))
			if result.Error != nil {
				t.Fatalf("sim error: %s", result.Error.Message)
			}
			dps := result.RaidMetrics.Parties[0].Players[0].Dps
			got := dps.Avg
			diff := (got - want.DPS) / want.DPS
			t.Logf("%s: ours %.1f ±%.1f, WarriorSim %.1f ±%.1f, difference %+.2f%% (tolerance %.1f%%)",
				sc.Name, got, dps.Stdev/math.Sqrt(iterations), want.DPS, want.DPSStderr, diff*100, sc.Tolerance*100)
			for _, action := range result.RaidMetrics.Parties[0].Players[0].Actions {
				damage := 0.0
				casts := int32(0)
				for _, tgt := range action.Targets {
					damage += tgt.Damage
					casts += tgt.Casts
				}
				if damage > 0 {
					t.Logf("  ours: %v dps %.1f casts %.1f", action.Id, damage/float64(iterations)/sc.FightSeconds, float64(casts)/float64(iterations))
				}
			}
			for name, a := range want.Abilities {
				t.Logf("  WarriorSim: %s dps %.1f casts %.1f", name, a.DPS, a.Casts)
			}
			if math.Abs(diff) > sc.Tolerance {
				t.Errorf("DPS differs from WarriorSim by %+.2f%%, more than the %.1f%% tolerance", diff*100, sc.Tolerance*100)
			}
		})
	}
	_ = fmt.Sprint
}
