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
| Warrior | Protection | Tank | Alpha | Yes | **Mechanics verified** | Threat of each ability matches in-game measurements (classic-warrior wiki, patch 1.13.6) exactly; the boss's hit table and damage taken match hand calculations (within 3 standard errors and 0.5%); see `validation/warrior/tank_test.go` | UNCERTAIN: Shield Slam adds block value once (Era tooltip; no in-game test). Rotation not compared with another sim. Battle Shout threat assumes a full party of 5. Set bonuses not checked. |
| Rogue | Combat / Daggers | DPS | Alpha | Yes | Not started | – | – |
| Mage | Fire / Frost | DPS | Alpha | Yes | Not started | – | – |
| Warlock | Destruction / Affliction | DPS | Alpha | Yes | Not started | – | – |
| Hunter | Marksmanship / Beast Mastery | DPS | Alpha | Yes | Not started | – | – |
| Priest | Shadow | DPS | Alpha | Yes | Not started | – | Meditation now gives the vanilla 15% casting regen (was 50%), so mana-limited fights read about 12% lower than before. |
| Priest | Holy | Healer | Alpha | Yes | **Mechanics verified** | Heal sizes, crits, mana costs, cast times, max mana and regen match hand calculations from the 1.12 spell data (within 0.5%, most within 0.1%); see `validation/priest` | HPS is raw output: no overhealing or raid damage yet. UNCERTAIN: Prayer of Healing +healing coefficient (0.286, community tables) and base mana (engine 1,376; 1.12 server table says 1,436). Power Word: Shield, Holy Nova, Lightwell not implemented. Discipline not started. |
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
- Boss swings used a Wrath of the Lich King attack power formula that added about 30% to a level 63 boss's hits. Now AP / 14 x swing time, as for players.
- The default incoming healing on tank pages (also from Wrath) was far below what a vanilla tank takes, so chance of death read 100%.
- Priest heals were commented-out Wrath of the Lich King code. Now the 1.12 ranks and coefficients.
- Meditation let 50% of spirit regen continue while casting (a later-expansion value); vanilla is 15%. Mental Strength raised Intellect; vanilla raises maximum mana by 10%. Force of Will no longer boosts heals.
- Shield Slam used the Season of Discovery version: twice the block value, extra attack power scaling and double threat. Now the Era version.

## Other versions

| Version | Status |
| --- | --- |
| WoW: Forever | Not started; research once it releases |
