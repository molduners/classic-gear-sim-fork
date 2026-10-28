import { IndividualSimUI, registerSpecConfig } from '../core/individual_sim_ui.js';
import { Player } from '../core/player.js';
import { APLRotation } from '../core/proto/apl.js';
import { Class, Faction, PartyBuffs, Race, Spec, Stat } from '../core/proto/common.js';
import { Stats } from '../core/proto_utils/stats.js';
import { getSpecIcon } from '../core/proto_utils/utils.js';
import * as HealingPriestInputs from './inputs.js';
import * as Presets from './presets.js';

const SPEC_CONFIG = registerSpecConfig(Spec.SpecHealingPriest, {
	cssClass: 'healing-priest-sim-ui',
	cssScheme: 'priest',
	// List any known bugs / issues here and they'll be shown on the site.
	knownIssues: [
		'Heals land in full: overhealing and a simulated raid taking damage are not modelled yet, so HPS is raw output.',
		'Power Word: Shield, Holy Nova, Lightwell and Inspiration\'s armor buff are not implemented yet.',
		'Prayer of Healing only heals the players in your party in the sim (just you, on this page).',
	],

	// All stats for which EP should be calculated.
	epStats: [Stat.StatIntellect, Stat.StatSpirit, Stat.StatHealingPower, Stat.StatSpellPower, Stat.StatSpellCrit, Stat.StatMP5],
	// Reference stat against which to calculate EP. I think all classes use either spell power or attack power.
	epReferenceStat: Stat.StatHealingPower,
	// Which stats to display in the Character Stats section, at the bottom of the left-hand sidebar.
	displayStats: [Stat.StatMana, Stat.StatIntellect, Stat.StatSpirit, Stat.StatHealingPower, Stat.StatSpellPower, Stat.StatSpellCrit, Stat.StatMP5],
	displayPseudoStats: [],

	defaults: {
		// Default equipped gear.
		gear: Presets.GearHolyP0.gear,
		// Default EP weights for sorting gear in the gear picker.
		epWeights: Stats.fromMap({
			// Rough starting weights until the sim computes its own.
			[Stat.StatIntellect]: 0.5,
			[Stat.StatSpirit]: 0.5,
			[Stat.StatHealingPower]: 1,
			[Stat.StatSpellPower]: 1,
			[Stat.StatSpellCrit]: 7,
			[Stat.StatMP5]: 2.2,
		}),
		// Default consumes settings.
		consumes: Presets.DefaultConsumes,
		// Default talents.
		talents: Presets.HolyTalents.data,
		// Default spec-specific settings.
		specOptions: Presets.DefaultOptions,
		// Default raid/party buffs settings.
		raidBuffs: Presets.DefaultRaidBuffs,
		partyBuffs: PartyBuffs.create({}),
		individualBuffs: Presets.DefaultIndividualBuffs,
		debuffs: Presets.DefaultDebuffs,
	},

	// IconInputs to include in the 'Player' section on the settings tab.
	playerIconInputs: [HealingPriestInputs.SelfPowerInfusion],
	// Buff and Debuff inputs to include/exclude, overriding the EP-based defaults.
	includeBuffDebuffInputs: [],
	excludeBuffDebuffInputs: [],
	// Inputs to include in the 'Other' section on the settings tab.
	otherInputs: {
		inputs: [],
	},
	encounterPicker: {
		// Whether to include 'Execute Duration (%)' in the 'Encounter' section of the settings tab.
		showExecuteProportion: false,
	},

	presets: {
		// Preset talents that the user can quickly select.
		talents: [Presets.HolyTalents],
		// Preset rotations that the user can quickly select.
		rotations: [Presets.ROTATION_PRESET_HOLY],
		// Preset gear configurations that the user can quickly select.
		gear: [Presets.GearHolyP0, Presets.GearBlank],
	},

	autoRotation: (_player: Player<Spec.SpecHealingPriest>): APLRotation => {
		return Presets.ROTATION_PRESET_HOLY.rotation.rotation!;
	},

	raidSimPresets: [
		{
			spec: Spec.SpecHealingPriest,
			tooltip: 'Holy Priest',
			defaultName: 'Holy',
			iconUrl: getSpecIcon(Class.ClassPriest, 1),

			talents: Presets.HolyTalents.data,
			specOptions: Presets.DefaultOptions,
			consumes: Presets.DefaultConsumes,
			defaultFactionRaces: {
				[Faction.Unknown]: Race.RaceUnknown,
				[Faction.Alliance]: Race.RaceDwarf,
				[Faction.Horde]: Race.RaceUndead,
			},
			defaultGear: {
				[Faction.Unknown]: {},
				[Faction.Alliance]: {
					1: Presets.GearHolyP0.gear,
				},
				[Faction.Horde]: {
					1: Presets.GearHolyP0.gear,
				},
			},
		},
	],
});

export class HealingPriestSimUI extends IndividualSimUI<Spec.SpecHealingPriest> {
	constructor(parentElem: HTMLElement, player: Player<Spec.SpecHealingPriest>) {
		super(parentElem, player, SPEC_CONFIG);
	}
}
