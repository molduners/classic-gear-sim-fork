import { IndividualSimUI } from '../../../individual_sim_ui';
import { EquipmentSpec, ItemSpec, Spec } from '../../../proto/common';
import { Database } from '../../../proto_utils/database';
import { BulkTab } from '../bulk_tab';
import { IndividualImporter } from './individual_importer';

export class BulkGearJsonImporter<SpecType extends Spec> extends IndividualImporter<SpecType> {
	private readonly bulkUI: BulkTab;

	constructor(parent: HTMLElement, simUI: IndividualSimUI<SpecType>, bulkUI: BulkTab) {
		super(parent, simUI, { title: 'Bag Item Import', allowFileUpload: true });

		this.bulkUI = bulkUI;
		this.descriptionElem.appendChild(
			<>
				<p>Import bag items from a JSON file, which can be created by the WowSimsExporter in-game AddOn.</p>
				<p>To import, upload the file or paste the text below, then click, 'Import'.</p>
			</>,
		);
	}

	async onImport(data: string) {
		try {
			const items = await parseBagExport(data);
			if (items.length > 0) {
				this.bulkUI.addItems(items);
			}
			this.close();
		} catch (e: any) {
			console.warn(e);
			alert(e.toString());
		}
	}
}

// Parses the WoWSims Exporter addon's bag export. Returns the items the sim's database knows.
export async function parseBagExport(data: string): Promise<Array<ItemSpec>> {
	const equipment = EquipmentSpec.fromJsonString(data, { ignoreUnknownFields: true });
	if (!(equipment?.items?.length > 0)) {
		return [];
	}
	const db = await Database.loadLeftoversIfNecessary(equipment);
	return equipment.items.filter(spec => spec.id > 0 && db.lookupItemSpec(spec));
}
