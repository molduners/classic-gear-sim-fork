package vanilla

import (
	"flag"
	"os"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/ruleset"
	"google.golang.org/protobuf/encoding/protojson"
	googleProto "google.golang.org/protobuf/proto"
)

var writeExample = flag.Bool("write-example", false, "rewrite examples/vanilla_fury_warrior.json")

const examplePath = "../../../examples/vanilla_fury_warrior.json"

func furyWarriorRequest() *proto.RaidSimRequest {
	gear := core.GetGearSet("../../../ui/warrior/gear_sets", "phase_1")
	apl := core.GetAplRotation("../../../ui/warrior/apls", "dps_reck")
	return &proto.RaidSimRequest{
		Raid: core.SinglePlayerRaidProto(
			core.WithSpec(&proto.Player{
				Name:          "Example Warrior",
				Race:          proto.Race_RaceOrc,
				Class:         proto.Class_ClassWarrior,
				Equipment:     gear.GearSet,
				TalentsString: "30305001302-05050005525010051",
				Consumes: &proto.Consumes{
					AgilityElixir:     proto.AgilityElixir_ElixirOfTheMongoose,
					AttackPowerBuff:   proto.AttackPowerBuff_JujuMight,
					DefaultPotion:     proto.Potions_MightyRagePotion,
					DragonBreathChili: true,
					Food:              proto.Food_FoodSmokedDesertDumpling,
					MainHandImbue:     proto.WeaponImbue_Windfury,
					OffHandImbue:      proto.WeaponImbue_ElementalSharpeningStone,
					StrengthBuff:      proto.StrengthBuff_JujuPower,
				},
				Buffs:              core.FullIndividualBuffs,
				Rotation:           apl.Rotation,
				DistanceFromTarget: 5,
				ReactionTimeMs:     150,
			}, &proto.Player_Warrior{Warrior: &proto.Warrior{Options: &proto.Warrior_Options{
				StartingRage: 50,
				Shout:        proto.WarriorShout_WarriorShoutBattle,
			}}}),
			core.FullPartyBuffs, core.FullRaidBuffs, core.FullDebuffs),
		Encounter: core.MakeSingleTargetEncounter(0),
		SimOptions: &proto.SimOptions{
			Iterations: 1000,
			RandomSeed: 101,
		},
	}
}

func TestVanillaIsRegistered(t *testing.T) {
	r, err := ruleset.Get(ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.MaxLevel != 60 {
		t.Fatalf("MaxLevel = %d, want 60", r.MaxLevel)
	}
}

// Activating the vanilla ruleset must register the spec modules so a full
// sim runs end to end, and a fixed seed must give identical results.
func TestVanillaRunsDeterministically(t *testing.T) {
	if err := ruleset.Activate(ID); err != nil {
		t.Fatal(err)
	}
	req := furyWarriorRequest()

	if *writeExample {
		out, err := protojson.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(req)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(examplePath, append(out, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	req.SimOptions = &proto.SimOptions{Iterations: 50, RandomSeed: 101, IsTest: true}
	a := core.RunRaidSim(req)
	b := core.RunRaidSim(googleProto.Clone(req).(*proto.RaidSimRequest))
	if a.Error != nil {
		t.Fatalf("sim error: %v", a.Error.Message)
	}
	dpsA := a.RaidMetrics.Dps.Avg
	if dpsA <= 0 {
		t.Fatalf("expected positive DPS, got %v", dpsA)
	}
	if dpsB := b.RaidMetrics.Dps.Avg; dpsA != dpsB {
		t.Fatalf("same seed gave different DPS: %v vs %v", dpsA, dpsB)
	}
}
