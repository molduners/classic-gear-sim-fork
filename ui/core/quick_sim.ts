// Quick Sim: paste a WoWSims Exporter addon string on the homepage, land on the right spec page,
// import the character over that page's default settings and run the sim straight away.
// Kept free of heavy imports so the homepage stays small.

const STORAGE_KEY = 'quickSim__pendingImport';
const URL_PARAM = 'quicksim';

export type QuickSimTarget = {
	// Folder of the spec's sim page, e.g. "warrior".
	page: string;
	// What we tell the user, e.g. "Fury Warrior".
	label: string;
	// True when the spec matches an independent reference (see COVERAGE.md).
	validated: boolean;
};

type ClassEntry = {
	name: string;
	// One entry per talent tree, in game order. null = no sim for specs that put most points there.
	trees: Array<{ spec: string; page: string; validated?: boolean } | null>;
};

const CLASSES: Record<string, ClassEntry> = {
	warrior: {
		name: 'Warrior',
		trees: [
			{ spec: 'Arms', page: 'warrior', validated: true },
			{ spec: 'Fury', page: 'warrior', validated: true },
			{ spec: 'Protection', page: 'tank_warrior' },
		],
	},
	rogue: {
		name: 'Rogue',
		trees: [
			{ spec: 'Assassination', page: 'rogue' },
			{ spec: 'Combat', page: 'rogue' },
			{ spec: 'Subtlety', page: 'rogue' },
		],
	},
	hunter: {
		name: 'Hunter',
		trees: [
			{ spec: 'Beast Mastery', page: 'hunter' },
			{ spec: 'Marksmanship', page: 'hunter' },
			{ spec: 'Survival', page: 'hunter' },
		],
	},
	mage: {
		name: 'Mage',
		trees: [
			{ spec: 'Arcane', page: 'mage' },
			{ spec: 'Fire', page: 'mage' },
			{ spec: 'Frost', page: 'mage' },
		],
	},
	warlock: {
		name: 'Warlock',
		trees: [
			{ spec: 'Affliction', page: 'warlock' },
			{ spec: 'Demonology', page: 'warlock' },
			{ spec: 'Destruction', page: 'warlock' },
		],
	},
	priest: {
		name: 'Priest',
		trees: [null, null, { spec: 'Shadow', page: 'shadow_priest' }],
	},
	druid: {
		name: 'Druid',
		trees: [{ spec: 'Balance', page: 'balance_druid' }, { spec: 'Feral', page: 'feral_druid' }, null],
	},
	shaman: {
		name: 'Shaman',
		trees: [{ spec: 'Elemental', page: 'elemental_shaman' }, { spec: 'Enhancement', page: 'enhancement_shaman' }, null],
	},
	paladin: {
		name: 'Paladin',
		trees: [null, { spec: 'Protection', page: 'protection_paladin' }, { spec: 'Retribution', page: 'retribution_paladin' }],
	},
};

// Points spent in each tree, from a talent string like "30305001302-05050005525010051-".
export function talentPointsPerTree(talents: string): Array<number> {
	return talents.split('-').map(tree => [...tree].reduce((sum, c) => sum + (Number(c) || 0), 0));
}

// Works out which sim page an exporter string belongs to. Throws an Error with a
// message meant for the user when it can't.
export function findQuickSimTarget(data: string): QuickSimTarget {
	let json: any;
	try {
		json = JSON.parse(data.trim());
	} catch {
		throw new Error("That doesn't look like an exporter string. Copy the whole text from the WoWSims Exporter addon window.");
	}
	const entry = CLASSES[String(json?.class ?? '').toLowerCase()];
	if (!entry) {
		throw new Error(`Couldn't tell the class from that string (got "${json?.class ?? ''}").`);
	}
	if (!json.gear || !Array.isArray(json.gear.items)) {
		throw new Error('That string has no gear in it. Export again from the addon.');
	}

	const points = talentPointsPerTree(String(json.talents ?? ''));
	let best = 0;
	points.forEach((p, i) => {
		if (p > (points[best] ?? 0)) best = i;
	});
	const tree = entry.trees[best] ?? entry.trees.find(t => t != null)!;
	if (!entry.trees[best] && points.some(p => p > 0)) {
		// e.g. a holy priest: the sim has no page for this spec yet.
		const known = entry.trees.filter(t => t != null).map(t => t!.spec);
		throw new Error(`The sim doesn't support that ${entry.name} spec yet. Supported: ${known.join(', ')}.`);
	}
	return { page: tree.page, label: `${tree.spec} ${entry.name}`, validated: !!tree.validated };
}

// Saves the string and returns the URL of the spec page that will pick it up.
export function startQuickSim(data: string, target: QuickSimTarget, basePath: string): string {
	window.sessionStorage.setItem(STORAGE_KEY, data.trim());
	return `${basePath}${target.page}/?${URL_PARAM}`;
}

// On a spec page: returns the pending exporter string, if this page was opened by Quick Sim.
// Only returns it once, and clears the URL flag so a reload doesn't re-import.
export function takeQuickSimImport(location: Location): string | null {
	if (!new URLSearchParams(location.search).has(URL_PARAM)) return null;
	const data = window.sessionStorage.getItem(STORAGE_KEY);
	window.sessionStorage.removeItem(STORAGE_KEY);
	window.history.replaceState(null, '', location.pathname + location.hash);
	return data;
}
