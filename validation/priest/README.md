# Holy Priest checks

`holy_test.go` checks the Holy Priest against hand calculations made from the 1.12 spell
data (cmangos classic-db `spell_template`) and the 1.12 server formulas (vmangos):

| Check | How it's worked out | Result |
| --- | --- | --- |
| Heal, Greater Heal, Flash Heal, Prayer of Healing size | (rank average + level scaling + coefficient × +healing) × Spiritual Healing | within 0.1% |
| Crits | 150% of a normal heal; chance = sheet crit + Holy Specialization | within 0.1%, within 3 standard errors |
| Spells learned below level 20 | coefficient × (1 − (20 − spell level) × 0.0375) | within 0.1% |
| Renew | per tick: rank amount + 20% of +healing, × (Spiritual Healing + Improved Renew), never crits | exact |
| Mana per cast | rank cost × Improved Healing / Mental Agility | exact |
| Cast time | Divine Fury takes 0.5s off Heal and Greater Heal | exact |
| Max mana and Mental Strength | base + 15 per Intellect after 20, × 1.10 for 5/5 Mental Strength | exact |
| Regen | 12.5 + Spirit/4 per 2s tick, 15% of it while casting (Meditation), plus mp5 | exact |

Coefficients: cast time / 3.5 for direct heals, duration / 15 for Renew. Prayer of Healing
uses 0.286 (a third of 3/3.5) from community tables; the server default would be 0.857, so
this one is marked UNCERTAIN in `sim/priest/heals.go`.

Run with `go test -tags=with_db ./validation/priest/ -v`.
