# Classic WoW Gear Sim

A Raidbots-style Quick Sim and Top Gear for World of Warcraft Classic, built on a
fork of [WoWSims Classic](https://github.com/wowsims/classic) (MIT). The owner is
not a developer: explain things in plain English, keep steps small, and ask one
decision at a time.

## Current milestone

**Milestone 5: tanking and healing.** First part is Protection Warrior: threat, the boss's
hit table and damage taken are checked in `validation/warrior/tank_test.go`, and the tank
rotation is `ui/tank_warrior/apls/prot.apl.json` (merged). Second part: vanilla-sized boss
swings and default incoming healing (`EnemyWeaponDamage`, `Player.setDefaultHealingParams`),
then the healer specs. Holy Priest is first (`sim/priest/heals.go`, `ui/healing_priest`,
checks in `validation/priest`); the healer view is "output and mana" for now (raw HPS and
time to out-of-mana), with a simulated raid and overhealing to come.

Merged so far: 1 (fork, ruleset layer), 2 (Warrior validated against WarriorSim,
`validation/warrior`), 3 (Quick Sim page, GitHub Pages) and 4 (Top Gear: the spec page's
Top Gear tab ranks bag combos with a ± margin and a "Too close to call" flag; Quick Sim
takes an optional bag export and opens straight into Top Gear).

The full plan (architecture, milestones, coverage) is in the project's plan doc:
https://claude.ai/code/artifact/61044d30-999a-4b64-9abf-ae7180a84baa

## Decisions

- Target vanilla Classic Era (level 60, nine classes) first. WoW: Forever and other
  versions come later as separate rulesets; don't assume anything about Forever.
- Fork and extend WoWSims rather than writing a new engine.
- Keep the WoWSims stack: Go engine, protobuf, TypeScript + Vite UI, WebAssembly in the browser.
- The site must keep a user-visible link back to WoWSims (their README asks for it).
- Warden Shaman (a Season of Discovery spec) is removed from the sim and hidden in the
  UI. Its protobuf messages remain so saved links and the UI type maps keep working.
- First spec to validate fully: Warrior (Arms/Fury).

## Architecture

- `sim/core` – shared engine: event loop, attack table, spells, auras, APL rotations, bulk sim.
- `sim/<class>/<spec>` – one module per class and spec.
- `sim/ruleset` – the seam between engine and game version. A `Ruleset` registers the
  spec modules for its version. The CLI chooses one with `--ruleset` (default `vanilla`).
- `sim/ruleset/vanilla` – Classic Era ruleset.
- `proto/` – shared data formats for engine and UI. `ui/` – web UI, one folder per spec.
- `assets/database` – generated item, enchant and spell data (see `docs/database_updates.md`).
- Quick Sim: `ui/homepage/quick_sim_box.ts` (homepage form) and `ui/core/quick_sim.ts` (picks the
  spec page from class + talent points, hands the string over through sessionStorage and a
  `?quicksim` flag). `IndividualSimUI.runQuickSim` imports it and runs the sim. Mark a spec
  `validated: true` there once it is validated in `COVERAGE.md`.
- Top Gear: `sim/core/bulksim.go` runs every combo with the same seed as the equipped gear
  and reports `dps_delta` ± `dps_delta_stderr` from the run-by-run differences. Fast mode
  starts at few iterations and keeps any combo within 3 standard errors of the 30th best,
  doubling until 30 remain. UI: `bulk_tab.tsx` and `bulk/bulk_sim_result_renderer.tsx`.
  `validation/warrior/top_gear_test.go` checks it against plain sims. The whole bulk sim
  runs in one web worker, so large bags are slow in the browser.

### Ruleset layer: what is and isn't behind it yet

Only spec registration goes through the ruleset today. Version-specific constants
(`sim/core/constants.go`, combat table maths, base stats) and the item database are
still vanilla values in `sim/core` and `assets/database`. Move each one behind the
`Ruleset` when a second version actually needs a different value, with the golden
tests proving vanilla results don't change.

## Commands

One-time setup (Go ≥ 1.23 and Node ≥ 20 installed):

```sh
npm ci
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6
export PATH=$PATH:$(go env GOPATH)/bin:$PWD/node_modules/@protobuf-ts/protoc/installed/protoc-36.2-linux-x86_64/bin
npx protoc --version   # downloads protoc on first run
make proto
```

Day to day:

```sh
# All engine tests (the item database needs the with_db tag)
go test -tags=with_db $(go list -e ./sim/... | grep -v sim/web)

# Re-record golden results after an intended change (then review the diff!)
make update-tests

# Run a sim from the command line (--summary prints DPS and a damage breakdown instead of JSON)
go run -tags=with_db ./cmd/wowsimcli sim --summary --infile examples/vanilla_fury_warrior.json

# Check the warrior against WarriorSim (details in validation/warrior/README.md)
go test -tags=with_db -v ./validation/warrior

# Build the website into dist/classic (don't use make -j, the asset rules race)
make
```

To view the built site: `cd dist && python3 -m http.server 8080`, then open
http://localhost:8080/classic/. Test exporter string: `examples/exporter_fury_warrior.json`.

The site has `/classic/` written into its HTML, CSS and JS. `tools/rebase_site.sh` moves a
built site to another path (GitHub Pages serves it at `/<repo name>/`).

## GitHub

- Repo: https://github.com/molduners/classic-gear-sim-fork (a copy of wowsims/classic with full history, not a GitHub fork).
- CI: `run_tests.yml` builds and runs the tests on every pull request to `master`.
- `sim.yml` ("Run a sim") runs one sim from the Actions tab, no local setup needed.
- `deploy.yml` publishes the site to GitHub Pages on every merge to `master`:
  https://molduners.github.io/classic-gear-sim-fork/ (needs Settings → Pages → Source: GitHub Actions).
- Removed upstream workflows that deploy to the wowsims site, post to their Discord, or cut weekly releases.

## Testing rules

- Golden tests (`*.results` files) are fixed-seed regression snapshots, not proof of
  accuracy. A spec is only "validated" in `COVERAGE.md` once it matches an independent
  reference within a recorded tolerance.
- Validation scenarios live in `validation/<class>/`. Never copy code from unlicensed
  reference sims (e.g. WarriorSim); fetch them into a gitignored `.cache/` instead.
- Tank pages put the player in front of the boss (`InFrontOfTarget`), so the boss can parry
  and block. Set it in tank tests too, and list the warrior in `Raid.tanks`.
- In APLs, queue Heroic Strike / Cleave / Maul with `"tag": 1`. Untagged, it casts as an
  instant extra attack and overstates DPS by about 30%.
- Every spell coefficient or proc rate we add or change gets a source note in a comment.
  Mark uncertain values `// UNCERTAIN:` and list them in `COVERAGE.md`.
