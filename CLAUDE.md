# Classic WoW Gear Sim

A Raidbots-style Quick Sim and Top Gear for World of Warcraft Classic, built on a
fork of [WoWSims Classic](https://github.com/wowsims/classic) (MIT). The owner is
not a developer: explain things in plain English, keep steps small, and ask one
decision at a time.

## Current milestone

**Milestone 1: fork and ruleset layer** (done locally, waiting for the GitHub fork).
Next: milestone 2, Warrior (Arms/Fury) end to end through the CLI, validated
against an independent reference.

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

# Run a sim from the command line
go run -tags=with_db ./cmd/wowsimcli sim --infile examples/vanilla_fury_warrior.json

# Build the website into dist/classic (don't use make -j, the asset rules race)
make
```

To view the built site: `cd dist && python3 -m http.server 8080`, then open
http://localhost:8080/classic/.

## GitHub

- Repo: https://github.com/molduners/classic-gear-sim-fork (a copy of wowsims/classic with full history, not a GitHub fork).
- CI: `run_tests.yml` builds and runs the tests on every pull request to `master`.
- Removed upstream workflows that deploy to the wowsims site, post to their Discord, or cut weekly releases.
  Hosting our own site is a later decision.

## Testing rules

- Golden tests (`*.results` files) are fixed-seed regression snapshots, not proof of
  accuracy. A spec is only "validated" in `COVERAGE.md` once it matches an independent
  reference within a recorded tolerance.
- Every spell coefficient or proc rate we add or change gets a source note in a comment.
  Mark uncertain values `// UNCERTAIN:` and list them in `COVERAGE.md`.
