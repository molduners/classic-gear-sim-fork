package encounters

import (
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/stats"
)

func addLevel60(bossPrefix string) {
	core.AddPresetTarget(&core.PresetTarget{
		PathPrefix: bossPrefix,
		Config: &proto.Target{
			Id:        213336, // TODO:
			Name:      "Level 60",
			Level:     63,
			MobType:   proto.MobType_MobTypeUnknown,
			TankIndex: 0,

			Stats: stats.Stats{
				stats.Health:      127_393, // TODO:
				stats.Armor:       3731,    // TODO:
				stats.AttackPower: 805,     // TODO:
			}.ToFloatArray(),

			SpellSchool: proto.SpellSchool_SpellSchoolPhysical,
			// A mid-tier raid boss. In the 1.12 emulator database (cmangos classic-db) Molten Core
			// bosses swing for about 1,600-2,900 and Blackwing Lair bosses about 2,400-5,100 before armour.
			// UNCERTAIN: not checked against Warcraft Logs.
			SwingSpeed:       2,
			MinBaseDamage:    3000,
			DamageSpread:     0.3333, // TODO:
			ParryHaste:       true,
			DualWield:        false,
			DualWieldPenalty: false,
			TargetInputs:     make([]*proto.TargetInput, 0),
		},
	})
	core.AddPresetEncounter("Level 60", []string{
		bossPrefix + "/Level 60",
	})
}
