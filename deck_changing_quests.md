# Deck-Changing Quests — Design Doc

## Context

Shandalar 30 is a roguelike deck-builder where the core engagement loop is **building and re-tuning your deck**. Today the only quests are the Wiseman's *delivery* and *defeat-enemy* quests (`game/domain/quest.go`), which test where you go and who you beat — not how you build. There's no incentive structure that nudges a player to crack open the deck editor and experiment with new colors, archetypes, or card pools.

This doc proposes a family of **action/constraint quests** — inspired by MTG Arena's daily quests but adapted to a roguelike — whose defining trait is that the most efficient way to complete them is to **change your deck**. "Cast 30 black or red spells" rewards a player who shifts toward those colors; "Win a duel with a deck of 20 cards or fewer" forces a genuine rebuild. The goal is to turn the deck editor from a once-per-session chore into a recurring, rewarded activity.

Per the user's direction, these quests **extend the existing Wiseman quest system** (new `QuestType`s, reusing its lifecycle/UI), span **both passive action-tracking and active deck-constraint** styles, and reward **gold, cards, amulets, and manalinks**.  Manalinks are no longer tied to a specific city.

Once these quests are completed they can be redeemed at any town. Walking into the town is enough to complete the quest.

Players can have up to 3 active quests at a time. Quest status is shown on the world frame using the Questnew.spr.png a sprite with 2 images, the first is shown if there are no active quests, the 2nd when there are active quests (it's an open scroll). The quests are printed on the scroll. It is positioned in the lower right of the world screen. Also, clicking on the scroll shows an overlay with the full quest description and progress to goal.

## Goals

- Add quests that reward deck experimentation and color/archetype diversity.
- Reuse the existing Wiseman quest giver, lifecycle, and reward sinks rather than building a parallel system.
- Define quests as **data (TOML)**, consistent with rogues and card tiers.
- Track in-duel actions accurately by tapping the engine's existing event system.

## Non-goals

- A separate "daily login" / real-time daily-reset system (these are in-game-day / Wiseman-gated).
- An achievements/history/meta-progression layer (no persistent completion log beyond the active quest).
- Changing the engine's rules or card behavior.

---

## Quest taxonomy

Two broad families, surfaced as new `QuestType` values.

### A. Passive action quests (track what you do across duels)

The player completes these just by playing, but the *efficient* path is to bias the deck toward the quest's theme. Progress accumulates across multiple duels until the target is hit or the quest expires.

| Quest | Tracks | Deck-change pressure |
|---|---|---|
| **Cast spells of color(s)** — "Cast 30 Black or Red spells" | `EvtSpellCast`, resolve card color | Pushes toward a specific 1–2 color identity |
| **Play lands** — "Play 40 lands" | `EvtLandPlayed` | Easiest / archetype-agnostic (the "safe" quest) |
| **Attack with creatures** — "Attack with 45 creatures" | `EvtDeclaredAttacker` | Pushes toward an aggressive creature deck |
| **Destroy opponent creatures** — "Destroy 20 of the enemy's creatures" | `EvtZoneChange` battlefield→graveyard on enemy creatures | Pushes toward removal / control |
| **Cast spells of a type** — "Cast 15 instants or sorceries" | `EvtSpellCast` + card type | Pushes toward spell-slinger builds |
| **Deal direct damage** — "Deal 30 damage to enemies with spells" | `EvtDamageDealt` (non-combat) | Pushes toward burn |

Counts accumulate on the active quest; progress persists across duels (and across win/loss) until the quest is fulfilled or expires.

### B. Active deck-constraint quests (validated at duel start; reward a win under the constraint)

These force a rebuild before you even fight. The constraint is checked against the submitted duel deck; the quest completes on a **win** while the constraint holds.

| Quest | Constraint checked against the deck |
|---|---|
| **Mono-color win** — "Win a duel with a mono-Green deck" | All non-land cards share one color |
| **Lean deck win** — "Win with a deck of N or fewer cards" | Deck size ≤ N (challenges `MinDeckSize` padding) |
| **Low-curve win** — "Win with only creatures of mana value ≤ 3" | Every creature's converted cost ≤ 3 |
| **Color-light win** — "Win a duel using no Blue cards" | No card of the named color in the deck |
| **Creatureless / spell win** — "Win without attacking" | No `EvtDeclaredAttacker` fired by the player that duel |

Constraint quests are higher-effort and pay the top reward tier.

---

## Player-facing flow

1. **Offer.** The Wiseman offers an action/constraint quest the same way it offers delivery/defeat quests today (`WisemanScreen`, `generateQuest()`). The offer screen states the objective, the target count/constraint, the deadline in days, and the reward tier.
2. **Hint to rebuild.** Constraint quests display a one-line nudge ("Edit your deck before you duel"). The city "Edit Deck" button already exists (`EditDeckScreen`).
3. **Progress.** Passive quests show `X / target` progress; it ticks up after each duel. Constraint quests show pass/fail of the deck check plus "win to complete."
4. **Completion & reward.** On fulfillment, the Wiseman pays out via existing reward sinks (gold / cards / amulets). Same `Reward` state the screen already has.
5. **Expiry.** Reuses the existing `DaysRemaining` countdown and `handleExpiredQuest()` (incl. the 20-day city quest ban on failure), so no new lifecycle is needed.

Reward tiers (mirroring MTGA's two-tier model, scaled to roguelike economy — exact numbers TBD in the tech design):

- **Standard** (easy passive, e.g. play lands): gold, scaled to player progression.
- **Themed** (color/type/aggro/control passive): more gold, or a small bundle of color-appropriate cards.
- **Challenge** (deck-constraint win): top gold + a card reward (consider `RandomPowerfulCardsForColor`) or amulets.

---

## Technical shape (enough to seed the tech design doc)

### Data model (s30)

- Extend `game/domain/quest.go`:
  - New `QuestType` values: `QuestTypeActionTracker`, `QuestTypeDeckConstraint`.
  - Add fields to `Quest`: an objective descriptor (metric kind + parameters like color mask / card type / count target), `Progress int`, and a `Constraint` descriptor for type-B quests.
  - Reuse existing `RewardType` (`RewardCard`, `RewardAmulet`, …) and `RewardCard`/`RewardAmulets` fields; add a gold reward field if not already representable.
- Define quests as **TOML** under `assets/configs/quests/` (precedent: `assets/configs/rogues/*.toml`, `card_tiers.toml`), loaded the way `rogue.go` loads rogues. Each entry: id, type, objective metric + target, eligible color/type params, deadline-days, reward tier. Generation in `wiseman.go` picks from this pool, scaled to player progression (reuse the scaling inputs in `world/enemy_spawn.go`).

### Action tracking — engine integration (mage-go)

The accurate signal lives in the engine, which s30 must not duplicate (engine code belongs in mage-go). Every game action funnels through **`Game.FireEvent`** (`pkg/mage/game.go:2013`), and there's a precedent observer in `pkg/mage/per_turn_trackers.go` (`recordPerTurnEvent`).

Two viable taps — pick one in the tech design:

1. **Preferred: per-duel objective counters in the engine.** Mirror the `per_turn_trackers.go` pattern with a small, *per-player, per-duel* aggregate (spells cast by color, lands played, attackers declared, enemy creatures destroyed, non-combat damage). Expose it through the existing `interactive.GameMsg` / end-of-duel result so s30 reads a compact tally when the duel ends. Clean attribution (spell color, exact attacker counts), minimal cross-module surface.
2. **Alternative: forward raw `GameEvent`s** to s30 via the duel channel and aggregate UI-side. More flexible, but widens the engine→UI contract.

Avoid the snapshot-diff approach (`duel.go:checkSoundTriggers`): it loses attribution (can't reliably get spell color or exact attacker counts).

**Event → metric mapping:**

- *Cast color/type*: `EvtSpellCast` → resolve `SourceID` → card → `ManaCost().Colors()` / `HasType(TypeCreature|TypeInstant|…)`. Note color-overridden cards need `Permanent.Colors()`.
- *Play lands*: `EvtLandPlayed`.
- *Attack*: `EvtDeclaredAttacker` (count per declaration).
- *Destroy enemy creature*: `EvtZoneChange` with `FromZone==Battlefield && ToZone==Graveyard`, source is a creature controlled by the opponent. (No dedicated death event; engine has an unexposed `creatureDeathsThisTurn`.)
- *Direct damage*: `EvtDamageDealt` (filter out combat damage).

### Deck-constraint validation (s30 only)

Validate the submitted duel deck at duel start, against `Player.GetDuelDeck()` (`domain/player.go`) — pure s30, no engine change:

- Mono-color / color-light: inspect each card's colors.
- Deck size: count cards (note `GetDuelDeck()` auto-pads to `MinDeckSize` with basics — validate pre-pad intent carefully).
- Curve: check each creature's mana value.
- "No attacking" is enforced via the action tracker (zero `EvtDeclaredAttacker` for the player), not a deck check.

If the constraint fails at duel start, surface a warning ("This deck doesn't meet the quest — duel won't count") but let the player duel anyway.

### Reward payout

Reuse the existing sinks — no new reward plumbing:

- Gold → `Player.Gold`
- Cards → `CardCollection.AddCard` (color bundles via `RandomPowerfulCardsForColor` in `card_tiers.go`)
- Amulets → `Player.AddAmulet`

Payout happens in `wiseman.go` `giveReward()`, extended to handle the new objective reward shapes.

---

## Open questions for the tech design

1. **Engine API boundary**: add a per-duel objective tally to `interactive.GameMsg`/end-result (preferred), vs. forwarding raw events. Settle the mage-go ↔ s30 contract.
2. **Progress persistence**: confirm `save/types.go` (`SaveData` → `*world.Level` → player) round-trips the new `Quest` fields (progress, objective, constraint). Should be automatic via the embedded player, but verify serialization.
3. **One-at-a-time vs. multiple**: today `Player.ActiveQuest` is a single quest. Do action quests share that slot, or do we allow a small queue (closer to MTGA's three dailies)? Recommend: keep single-slot for v1 to minimize lifecycle changes.
4. **Reward scaling numbers**: concrete gold/card amounts per tier, scaled to difficulty and progression.
5. **Color matching edge cases**: hybrid mana and color-overridden cards — confirm `Permanent.Colors()` is reachable at the count site.

## Verification plan

- **Unit (s30, TDD)**: quest generation from TOML; progress accumulation given a synthetic per-duel tally; deck-constraint validators against hand-built decks; reward payout mutates the right sinks. (`go test ./...`.)
- **Engine (mage-go)**: tests for the per-duel objective counter using `pkg/mage/gametest/`, asserting e.g. casting two red spells + one black increments the right buckets; an enemy creature dying increments the destroy counter.
- **End-to-end**: drive a real duel via `cmd/duel_test/main.go`, complete a passive quest across one or more duels, and confirm the Wiseman pays out. For a constraint quest, submit a conforming and a non-conforming deck and confirm validation + win-gated completion.
- **Manual**: accept each quest type from the Wiseman, watch progress tick, rebuild a deck to satisfy a constraint, collect the reward.

## Representative files to touch

- `game/domain/quest.go` — new quest types/fields.
- `game/screens/wiseman.go` — offer/track/reward for new types; `generateQuest()`, `giveReward()`.
- `game/screens/duel/duel.go` — read end-of-duel objective tally; apply to active quest.
- `assets/configs/quests/*.toml` (new) + a loader alongside `game/domain/rogue.go`.
- mage-go `pkg/mage/per_turn_trackers.go` / `interactive/types.go` — per-duel objective counters + expose via `GameMsg`/result.
- `game/save/types.go` — verify new quest fields persist.
