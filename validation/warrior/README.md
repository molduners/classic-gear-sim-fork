# Warrior validation against WarriorSim

Checks our Arms and Fury warrior against [WarriorSim](https://github.com/GuybrushGit/WarriorSim),
a separate, widely used vanilla warrior simulator written independently of WoWSims.
Both sims run the same scenarios (`scenarios.json`) and `reference_test.go` fails if
our DPS is further from WarriorSim's than the scenario's tolerance.

WarriorSim has no licence, so none of its code is copied here. `fetch_warriorsim.sh`
clones a pinned commit into `.cache/` (gitignored) and `run_warriorsim.mjs` runs it
in Node with a stubbed page.

## Commands

```sh
# Compare against the recorded WarriorSim numbers (runs in CI with the other tests)
go test -tags=with_db -v ./validation/warrior

# Re-record WarriorSim's numbers (needs Node 20 and network for the first fetch)
./validation/warrior/fetch_warriorsim.sh
node validation/warrior/run_warriorsim.mjs 20000            # all scenarios
node validation/warrior/run_warriorsim.mjs 20000 fury-dw-p1 # just one
```

## Results

WarriorSim commit `ae03455a`, 20,000 iterations; ours 5,000 iterations, seed 101.
Margin of error is under 0.3 DPS on both sides.

| Scenario | What it checks | Ours | WarriorSim | Difference | Tolerance |
| --- | --- | --- | --- | --- | --- |
| white-2h-tauren | Attack table, glancing, 2H white hits | 37.7 | 37.3 | +1.3% | 2% |
| white-2h-orc-axe | Racial weapon skill (305) | 43.2 | 43.2 | +0.1% | 2% |
| white-dw-tauren | Dual wield miss and off-hand damage | 35.3 | 35.0 | +0.8% | 2% |
| fury-dw-p1 | Fury rotation, phase 1 gear, no buffs | 184.0 | 180.4 | +2.0% | 3% |
| arms-2h-p1 | Arms rotation, phase 1 gear, no buffs | 148.2 | 146.6 | +1.1% | 3% |
| fury-dw-p1-buffed | Fury as Human with Kings, Might, MotW, LotP and consumables | 292.2 | 288.2 | +1.4% | 3% |

All scenarios: level 63 target with 3731 armour, 120 second fight, no execute phase.

## Known differences

These are where the two sims disagree. No primary source settles them; the first two
are marked `UNCERTAIN` in the code, and all are listed in `COVERAGE.md`.

- **Deep Wounds from off-hand crits.** We use half the off-hand weapon's damage;
  WarriorSim always uses the main hand. Our Fury Deep Wounds is about 20% lower.
- **Dual wield miss.** We add 19% (magey wiki); WarriorSim uses 0.8 × miss + 20%.
- **Crit suppression.** The two sims apply the level 63 boss's crit suppression in
  different places (WarriorSim's character sheet shows 1.8% less crit than ours).
- **Off-hand white damage** is about 5% higher in ours, and we cast slightly more
  Heroic Strikes. Both follow from the items above.

## Not covered yet

- The execute phase (all scenarios set it to 0%).
- Enchants, world buffs, weapon imbues, other trinkets and set bonuses.
- Protection warrior.

## Bugs this found

- Rage from damage used `^` (bitwise XOR in Go) where it meant "squared", which gave
  warriors about 16% too much rage. Fixed in `sim/core/rage.go`.
- Armour reduction had no 75% cap. Fixed in `sim/core/spell_resistances.go`.
