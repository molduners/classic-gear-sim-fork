// Runs each scenario in scenarios.json through WarriorSim
// (https://github.com/GuybrushGit/WarriorSim), an independent, widely used
// Classic warrior simulator, and writes the results to reference_results.json.
//
// WarriorSim has no licence file, so it is not copied into this repository:
// fetch_warriorsim.sh clones it at a pinned commit into .cache/WarriorSim.
//
// Usage: node run_warriorsim.mjs [iterations]
import fs from 'node:fs';
import path from 'node:path';
import vm from 'node:vm';
import { execSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const wsDir = path.join(here, '.cache', 'WarriorSim');
const iterations = parseInt(process.argv[2] || '20000');
const only = process.argv[3]; // optional: run a single scenario by name

const { scenarios } = JSON.parse(fs.readFileSync(path.join(here, 'scenarios.json'), 'utf8'));

// WarriorSim buff names for our neutral buff keys.
const BUFFS = {
  battle_stance: [2457],
  berserker_stance: [2458],
  kings: [20217],
  might_improved: [25291, 20048],
  motw_improved: [9885, 17055],
  leader_of_the_pack: [17007],
  mongoose: [13452],
  juju_power: [12451],
  juju_might: [12460],
  dumplings: [20452],
};

function loadWarriorSim() {
  // WarriorSim touches the page through jQuery in a few places; a do-nothing stub is enough.
  const stub = new Proxy(function () {}, { get: () => stub, apply: () => stub });
  const jq = stub;
  const ctx = { console, Math, Date, setTimeout, $: jq, window: { location: { href: 'classic.html' } } };
  ctx.globalThis = ctx;
  vm.createContext(ctx);
  for (const f of [
    'data/buffs.js', 'data/enchants.js', 'data/levelstats.js', 'data/spells.js', 'data/talents.js', 'data/gear.js',
    'classes/player.js', 'classes/simulation.js', 'classes/spell.js', 'classes/weapon.js', 'globals.js',
  ]) {
    vm.runInContext(fs.readFileSync(path.join(wsDir, 'js', f), 'utf8'), ctx, { filename: f });
  }
  return ctx;
}

// Talent strings use the in-game order: per tree, row by row, left to right.
function applyTalents(ctx, talentString) {
  const trees = vm.runInContext('talents', ctx);
  const parts = talentString ? talentString.split('-') : [];
  trees.forEach((tree, i) => {
    const ordered = [...tree.t].sort((a, b) => a.y - b.y || a.x - b.x);
    const digits = parts[i] || '';
    ordered.forEach((talent, j) => { talent.c = parseInt(digits[j] || '0'); });
  });
}

function applyGear(ctx, gearSpec) {
  const gear = vm.runInContext('gear', ctx);
  for (const type in gear) for (const item of gear[type]) item.selected = false;
  for (const [slot, id] of Object.entries(gearSpec)) {
    const candidates = slot === 'mainhand' ? ['mainhand', 'twohand'] : [slot];
    const item = candidates.flatMap(t => gear[t] || []).find(i => i.id === id);
    if (!item) throw new Error(`WarriorSim has no item ${id} for slot ${slot}`);
    item.selected = true;
  }
  const enchant = vm.runInContext('enchant', ctx);
  for (const type in enchant) for (const e of enchant[type]) e.selected = false;
}

function applyBuffs(ctx, keys) {
  const buffs = vm.runInContext('buffs', ctx);
  for (const b of buffs) b.active = false;
  for (const key of keys) {
    if (!BUFFS[key]) throw new Error(`unknown buff ${key}`);
    for (const id of BUFFS[key]) buffs.find(b => b.id === id).active = true;
  }
}

// WarriorSim spell IDs and settings for our neutral rotation names. These
// must match the APLs in reference_test.go.
const ROTATIONS = {
  white: [],
  fury: [
    { id: 2687, timetostart: 0, timetostartactive: true, timetoendactive: false }, // Bloodrage on cooldown
    { id: 23894, minrageactive: false }, // Bloodthirst
    { id: 1680, minrageactive: false, maxrageactive: false, maincdactive: false }, // Whirlwind
    { id: 25286, minrage: 30, minrageactive: true, unqueueactive: false, maincdactive: false }, // Heroic Strike at 30+ rage
  ],
  arms: [
    { id: 2687, timetostart: 0, timetostartactive: true, timetoendactive: false }, // Bloodrage on cooldown
    { id: 27580, minrageactive: false }, // Mortal Strike (160 bonus damage; WoWSims calls it 21553)
    { id: 1680, minrageactive: false, maxrageactive: false, maincdactive: false }, // Whirlwind
    { id: 25286, minrage: 50, minrageactive: true, unqueueactive: false, maincdactive: false }, // Heroic Strike at 50+ rage
  ],
};

function applyRotation(ctx, rotation, executePercent) {
  const spells = vm.runInContext('spells', ctx);
  for (const s of spells) s.active = false;
  const entries = ROTATIONS[rotation];
  if (!entries) throw new Error(`rotation ${rotation} not mapped yet`);
  for (const { id, ...props } of entries) {
    const spell = spells.find(s => s.id === id);
    if (!spell) throw new Error(`WarriorSim has no spell ${id}`);
    Object.assign(spell, props, { active: true });
  }
  if (executePercent > 0) throw new Error('execute phase not mapped yet');
}

function run(ctx, sc) {
  applyTalents(ctx, sc.talents);
  applyGear(ctx, sc.gear);
  applyBuffs(ctx, sc.buffs.length ? sc.buffs : ['battle_stance']);
  applyRotation(ctx, sc.rotation, sc.executePercent);
  ctx.__config = {
    level: 60, race: sc.race, aqbooks: true, // the sim uses AQ ranks (core.IncludeAQ)
    reactionmin: 0, reactionmax: 0, adjacent: 0,
    mode: 'classic', spellqueueing: false,
    target: { level: 63, basearmor: sc.targetArmor, defense: 315, resistance: 0, speed: 0, mindmg: 0, maxdmg: 0, bleedreduction: 1 },
  };
  ctx.__sim = {
    timesecsmin: sc.fightSeconds, timesecsmax: sc.fightSeconds, executeperc: sc.executePercent,
    startrage: 0, iterations, batching: 10, // 10 ms spell batching, the WarriorSim default (WoWSims: core.SpellBatchWindow)
  };
  return vm.runInContext(`(() => {
    let out;
    const player = new Player(null, null, null, __config);
    const sim = new Simulation(player, r => { out = r; }, null, __sim);
    sim.startSync();
    const n = out.iterations;
    const mean = out.sumdps / n;
    const sd = Math.sqrt(Math.max(out.sumdps2 / n - mean * mean, 0));
    const weapon = w => w && { totalDamage: w.totaldmg / n, outcomes: { hit: w.data[0], miss: w.data[1], dodge: w.data[2], crit: w.data[3], glance: w.data[4] } };
    const abilities = {};
    for (const [k, sp] of Object.entries(player.spells)) if (sp.totaldmg) abilities[k] = { dps: sp.totaldmg / n / sim.duration, casts: (sp.data || []).reduce((a, b) => a + b, 0) / n };
    for (const [k, au] of Object.entries(player.auras)) if (au.totaldmg) abilities[k] = { dps: au.totaldmg / n / sim.duration };
    abilities.mh = { dps: player.mh.totaldmg / n / sim.duration };
    if (player.oh) abilities.oh = { dps: player.oh.totaldmg / n / sim.duration };
    for (const w of [player.mh, player.oh]) if (w && w.totalprocdmg) abilities[(w === player.mh ? 'mh' : 'oh') + 'procs'] = { dps: w.totalprocdmg / n / sim.duration };
    return { iterations: n, dps: mean, abilities, dpsStdev: sd, dpsStderr: sd / Math.sqrt(n), mh: weapon(player.mh), oh: weapon(player.oh),
      stats: { ap: player.stats.ap, str: player.stats.str, agi: player.stats.agi, crit: player.crit, skillMH: player.stats['skill_' + player.mh.type] } };
  })()`, ctx);
}

const commit = execSync('git rev-parse HEAD', { cwd: wsDir }).toString().trim();
// Running one scenario updates just that entry and keeps the others.
const outFile = path.join(here, 'reference_results.json');
const previous = only && fs.existsSync(outFile) ? JSON.parse(fs.readFileSync(outFile, 'utf8')) : null;
const results = { warriorSimCommit: commit, scenarios: previous && previous.warriorSimCommit === commit ? previous.scenarios : {} };
for (const sc of scenarios.filter(s => !only || s.name === only)) {
  const ctx = loadWarriorSim();
  results.scenarios[sc.name] = run(ctx, sc);
  console.log(sc.name, results.scenarios[sc.name].dps.toFixed(1), '±', results.scenarios[sc.name].dpsStderr.toFixed(1));
}
fs.writeFileSync(outFile, JSON.stringify(results, null, 2) + '\n');
