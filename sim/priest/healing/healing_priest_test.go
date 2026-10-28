package healing

import (
	"testing"

	_ "github.com/wowsims/classic/sim/common" // imported to get item effects included.
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
)

func init() {
	RegisterHealingPriest()
}

func TestHoly(t *testing.T) {
	core.RunTestSuite(t, t.Name(), core.FullCharacterTestSuiteGenerator([]core.CharacterSuiteConfig{
		{
			Class:    proto.Class_ClassPriest,
			Phase:    1,
			Race:     proto.Race_RaceUndead,
			IsHealer: true,

			GearSet:     core.GetGearSet("../../../ui/healing_priest/gear_sets", "p0.holy"),
			Talents:     HolyTalents,
			Rotation:    core.GetAplRotation("../../../ui/healing_priest/apls", "holy"),
			Buffs:       core.FullBuffs,
			Consumes:    FullConsumes,
			SpecOptions: core.SpecOptionsCombo{Label: "Holy", SpecOptions: PlayerOptionsHoly},

			ItemFilter: core.ItemFilter{
				WeaponTypes: []proto.WeaponType{
					proto.WeaponType_WeaponTypeDagger,
					proto.WeaponType_WeaponTypeMace,
					proto.WeaponType_WeaponTypeOffHand,
					proto.WeaponType_WeaponTypeStaff,
				},
				ArmorType: proto.ArmorType_ArmorTypeCloth,
				RangedWeaponTypes: []proto.RangedWeaponType{
					proto.RangedWeaponType_RangedWeaponTypeWand,
				},
			},

			EPReferenceStat: proto.Stat_StatHealingPower,
			StatsToWeigh: []proto.Stat{
				proto.Stat_StatIntellect,
				proto.Stat_StatSpirit,
				proto.Stat_StatHealingPower,
				proto.Stat_StatSpellCrit,
				proto.Stat_StatMP5,
			},
		},
	}))
}

var HolyTalents = "05420013050-034050032300155"

var FullConsumes = core.ConsumesCombo{
	Label: "Mana-Consumes",
	Consumes: &proto.Consumes{
		Flask:           proto.Flask_FlaskOfDistilledWisdom,
		Food:            proto.Food_FoodNightfinSoup,
		ManaRegenElixir: proto.ManaRegenElixir_MagebloodPotion,
	},
}

var PlayerOptionsHoly = &proto.Player_HealingPriest{
	HealingPriest: &proto.HealingPriest{
		Options: &proto.HealingPriest_Options{},
	},
}
