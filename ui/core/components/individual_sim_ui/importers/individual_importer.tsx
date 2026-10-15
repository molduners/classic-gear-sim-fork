import { SimSettingCategories } from '../../../constants/sim_settings';
import { IndividualSimUI } from '../../../individual_sim_ui';
import { Class, EquipmentSpec, Profession, Race, Spec } from '../../../proto/common';
import { Database } from '../../../proto_utils/database';
import { classNames } from '../../../proto_utils/names';
import { TypedEvent } from '../../../typed_event';
import { getEnumValues } from '../../../utils';
import { Importer, ImporterOptions } from '../../importer';
import Toast from '../../toast';

// For now this just holds static helpers to match the exporter, so it doesn't extend Importer.
export abstract class IndividualImporter<SpecType extends Spec> extends Importer {
	// Exclude UISettings by default, since most users don't intend to export those.
	static readonly DEFAULT_CATEGORIES = getEnumValues(SimSettingCategories).filter(c => c !== SimSettingCategories.UISettings) as Array<SimSettingCategories>;
	static readonly CATEGORY_PARAM = 'i';

	protected readonly simUI: IndividualSimUI<any>;

	constructor(parent: HTMLElement, simUI: IndividualSimUI<SpecType>, options: ImporterOptions) {
		super(parent, options);
		this.simUI = simUI;
	}

	protected async finishIndividualImport<SpecType extends Spec>(
		simUI: IndividualSimUI<SpecType>,
		charClass: Class,
		race: Race,
		equipmentSpec: EquipmentSpec,
		talentsStr: string,
		professions: Array<Profession>,
	): Promise<void> {
		const missing = await applyCharacterImport(simUI, charClass, race, equipmentSpec, talentsStr, professions);
		this.close();
		showImportResultToast(missing);
	}
}

export type MissingImportIds = {
	items: Array<number>;
	enchants: Array<number>;
};

// Applies an imported character (race, gear, talents, professions) to the sim. Shared by the
// import modals and Quick Sim. Returns the item and enchant IDs the database doesn't know.
export async function applyCharacterImport<SpecType extends Spec>(
	simUI: IndividualSimUI<SpecType>,
	charClass: Class,
	race: Race,
	equipmentSpec: EquipmentSpec,
	talentsStr: string,
	professions: Array<Profession>,
): Promise<MissingImportIds> {
	const playerClass = simUI.player.getClass();
	if (charClass != playerClass) {
		throw new Error(`Wrong Class! Expected ${classNames.get(playerClass)} but found ${classNames.get(charClass)}!`);
	}

	await Database.loadLeftoversIfNecessary(equipmentSpec);

	const gear = simUI.sim.db.lookupEquipmentSpec(equipmentSpec);

	const expectedEnchantIds = equipmentSpec.items.map(item => item.enchant);
	const foundEnchantIds = gear.asSpec().items.map(item => item.enchant);
	const missingEnchants = expectedEnchantIds.filter(expectedId => expectedId != 0 && !foundEnchantIds.includes(expectedId));

	const expectedItemIds = equipmentSpec.items.map(item => item.id);
	const foundItemIds = gear.asSpec().items.map(item => item.id);
	const missingItems = expectedItemIds.filter(expectedId => !foundItemIds.includes(expectedId));

	// Now update settings using the parsed values.
	const eventID = TypedEvent.nextEventID();
	TypedEvent.freezeAllAndDo(() => {
		simUI.player.setRace(eventID, race);
		simUI.player.setGear(eventID, gear);
		if (talentsStr && talentsStr != '--') {
			simUI.player.setTalentsString(eventID, talentsStr);
		}
		if (professions.length > 0) {
			simUI.player.setProfessions(eventID, professions);
		}
	});

	return { items: missingItems, enchants: missingEnchants };
}

export function showImportResultToast(missing: MissingImportIds) {
	if (missing.items.length == 0 && missing.enchants.length == 0) {
		new Toast({ variant: 'success', body: `Import successful!` });
	} else {
		new Toast({
			variant: 'info',
			body:
				'Import successful, but the following IDs were not found in the sim database:' +
				(missing.items.length == 0 ? '' : '\n\nItems: ' + missing.items.join(', ')) +
				(missing.enchants.length == 0 ? '' : '\n\nEnchants: ' + missing.enchants.join(', ')),
		});
	}
}
