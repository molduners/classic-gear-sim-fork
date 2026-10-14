# Coverage matrix

What each class and spec can do, per game version. Nothing counts as supported until
its rotation is checked and it matches an independent reference (real logs,
community spreadsheets or hand calculations) within a recorded tolerance.

Status levels: **Not started** → **Rotation drafted** → **Mechanics verified** → **Validated** (against a reference).

"WoWSims label" is the upstream launch status we inherited. "Golden test" means a
fixed-seed regression snapshot exists; that catches changes, not inaccuracy.

## Vanilla (Classic Era)

| Class | Spec | Role | WoWSims label | Golden test | Our status | Reference / tolerance | Known approximations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Warrior | Arms / Fury | DPS | Alpha | Yes | **Validated** | WarriorSim, within 3% (all six scenarios within 2%); see `validation/warrior` | UNCERTAIN: Deep Wounds from off-hand crits, dual wield miss formula, where crit suppression applies. Execute phase not validated yet. |
| Warrior | Protection | Tank | Alpha | Yes | Not started | – | – |
| Rogue | Combat / Daggers | DPS | Alpha | Yes | Not started | – | – |
| Mage | Fire / Frost | DPS | Alpha | Yes | Not started | – | – |
| Warlock | Destruction / Affliction | DPS | Alpha | Yes | Not started | – | – |
| Hunter | Marksmanship / Beast Mastery | DPS | Alpha | Yes | Not started | – | – |
| Priest | Shadow | DPS | Alpha | Yes | Not started | – | – |
| Priest | Holy / Discipline | Healer | Off | No | Not started | – | No healing model yet |
| Druid | Balance | DPS | Beta | Yes | Not started | – | – |
| Druid | Feral (cat) | DPS | Alpha | Yes | Not started | – | – |
| Druid | Feral (bear) | Tank | Off | No | Not started | – | – |
| Druid | Restoration | Healer | Off | No | Not started | – | No healing model yet |
| Shaman | Elemental | DPS | Launched | Yes | Not started | – | – |
| Shaman | Enhancement | DPS | Launched | Yes | Not started | – | – |
| Shaman | Restoration | Healer | Off | No | Not started | – | No healing model yet |
| Paladin | Holy | Healer | Off | No | Not started | – | No healing model yet |
| Paladin | Protection | Tank | Alpha | Yes | Not started | – | – |
| Paladin | Retribution | DPS | Alpha | Yes | Not started | – | – |

## Engine fixes found by validation

- Rage from damage was about 16% too high (`^` used as "squared" in Go). Affects every rage user.
- Armour reduction now caps at 75%.

## Other versions

| Version | Status |
| --- | --- |
| WoW: Forever | Not started; research once it releases |
