import * as PresetUtils from '../core/preset_utils.js';
import {
	Consumes,
	Debuffs,
	Flask,
	Food,
	IndividualBuffs,
	RaidBuffs,
	TristateEffect,
	UnitReference,
} from '../core/proto/common.js';
import {
	HealingPriest_Options as Options,
} from '../core/proto/priest.js';
import { SavedTalents } from '../core/proto/ui.js';
import HolyApl from './apls/holy.apl.json';
import BlankGear from './gear_sets/blank.gear.json';
import HolyP0Gear from './gear_sets/p0.holy.gear.json';

// Preset options for this spec.
// Eventually we will import these values for the raid sim too, so its good to
// keep them in a separate file.

export const GearBlank = PresetUtils.makePresetGear('Blank', BlankGear);
export const GearHolyP0 = PresetUtils.makePresetGear('Pre-raid Holy', HolyP0Gear);

export const ROTATION_PRESET_HOLY = PresetUtils.makePresetAPLRotation('Holy', HolyApl);

// Default talents. Uses the wowhead calculator format, make the talents on
// https://wowhead.com/classic/talent-calc and copy the numbers in the url.
// 20/31/0 raid Holy: Meditation, Mental Agility, Inner Focus, Divine Fury, Improved Healing,
// Spiritual Guidance and Spiritual Healing.
export const HolyTalents = {
	name: 'Holy',
	data: SavedTalents.create({
		talentsString: '05420013050-034050032300155',
	}),
};

export const DefaultOptions = Options.create({
	useInnerFire: true,

	powerInfusionTarget: UnitReference.create(),
});

export const DefaultConsumes = Consumes.create({
	flask: Flask.FlaskUnknown,
	food: Food.FoodUnknown,
});

export const DefaultRaidBuffs = RaidBuffs.create({
	giftOfTheWild: TristateEffect.TristateEffectImproved,
	powerWordFortitude: TristateEffect.TristateEffectImproved,
	strengthOfEarthTotem: TristateEffect.TristateEffectRegular,
	arcaneBrilliance: true,
	divineSpirit: true,
	moonkinAura: true,
});

export const DefaultIndividualBuffs = IndividualBuffs.create({
	blessingOfKings: true,
	blessingOfWisdom: TristateEffect.TristateEffectImproved,
});

export const DefaultDebuffs = Debuffs.create({
});
