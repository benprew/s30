# Deck-Changing Quests — Implementation TODO

Tracking implementation of `deck_changing_quests.md`. Implemented in phases.

## Design decisions (from doc body, overriding "open questions")
- **Up to 3 active quests** at a time (not single-slot). `Player.ActiveQuest` → multi-quest.
- Quests redeemable at **any town** — walking in completes a fulfilled quest.
- World-frame quest status via `Questnew.spr.png` (2 frames: empty / open-scroll), lower-right.
- Clicking the scroll opens an overlay with full quest descriptions + progress.
- Quests defined as **TOML** under `assets/configs/quests/`.
- Manalinks no longer tied to a specific city.

## Phase 0 — Exploration & design (DONE)
- [x] Map quest lifecycle (wiseman, city, player, save)
- [x] Map world-frame / world-screen UI + sprites
- [x] Map duel engine event system (mage-go)
- [x] Engine boundary: per-duel counters on *Game (mirror per_turn_trackers.go),
      exposed via `DuelObjectivesFor(playerID) DuelObjectives`, read at GameOver in duel.go
- [x] Storage: add `Player.ActiveQuests []*Quest` (cap 3) for deck-changing quests;
      leave legacy `ActiveQuest *Quest` (delivery/defeat) untouched. Reuse Quest struct,
      RewardType/giveReward sinks, Wiseman offer screen, DaysRemaining countdown.

### Key facts found
- mage-go at /home/ben/src/mage-go; events fire through `Game.FireEvent` (game.go:2012).
- Per-turn precedent: `pkg/mage/per_turn_trackers.go` (recorder + lazy maps + accessors + reset).
- Event sites: EvtSpellCast (spell_cast_stack.go:64, SourceID=card on stack, PlayerID=caster);
  EvtLandPlayed (game.go:3942, PlayerID); EvtDeclaredAttacker (game.go:3465, PlayerID=active);
  EvtDamageDealt to player (game.go:1767, Flag=IsCombatDamage); creature death at 3 sites where
  `creatureDeathsThisTurn++` (game.go:1036/1118/1242) — record duel deaths inline there (handles tokens).
- Clone (clone.go) deep-copies per-turn maps via cloneUUIDMap — must clone per-duel maps too.
- Card colors: domain.Card.Colors []string + colorStringToMask; CardType enum strings; mana value via ManaCost parse.
- Save is JSON via Level.Player; compare cities by .Name (pointers don't survive round-trip).
- Sprite: Questnew.spr.png = 474x115 = 2 frames 237x115; NOT yet embedded.

## Phase 1 — Data model (s30) ✅ DONE
- [x] Extend `game/domain/quest.go`: new QuestTypes, QuestMetric/QuestConstraint, Progress, gold reward, helpers
- [x] `Card.ColorMask()` / `Card.ManaValue()` / `Card.IsLand()` helpers (card.go)
- [x] TOML schema + loader: `quest_def.go` + `assets/configs/quests/quests.toml` (13 quests, all kinds), embed added
- [x] `Player.ActiveQuests []*Quest` (cap 3) + Add/Has/Remove/Expire helpers; DaysRemaining countdown
- [x] Unit tests: quest_def_test.go, quest_test.go — all pass; full module builds

## Phase 2 — Deck-constraint validation (s30 only) ✅ DONE
- [x] `quest_validate.go`: ValidateDeckConstraint (mono-color, lean, low-curve, color-light; no-attacking = deck-agnostic)
- [x] Validates the UN-padded constructed deck (CardCollection.GetDeck), not GetDuelDeck (padding defeats lean/color-light)
- [x] Unit tests quest_validate_test.go — all pass
- Note: lean-deck N (20) is below MinDeckSize (30-40); intent measured on built deck per doc. Duel-start wiring in Phase 4.

## Phase 3 — Engine action tracking (mage-go) ✅ DONE
- [x] `pkg/mage/per_duel_trackers.go`: per-duel counters (color/type/lands/attackers/deaths/non-combat dmg), NOT reset per turn
- [x] `recordPerDuelEvent` wired into FireEvent; `recordCreatureDeath` at 3 death sites (game.go) — counts tokens
- [x] Accessor `(*Game).DuelObjectivesFor(playerID) DuelObjectives` (opponent-relative creature kills)
- [x] Deep-clone nested maps in clone.go (cloneUUIDColorMap/cloneUUIDCardTypeMap) — AI search can't corrupt counters
- [x] Engine tests `pkg/mage/gametest/per_duel_trackers_test.go` — all pass; full mage-go suite green
- API for s30: `s.game.DuelObjectivesFor(s.human.PlayerID())` read at GameOver. core.Color/core.CardType in result.

## Phase 4 — Quest progress wiring (s30 duel) ✅ DONE
- [x] `quest_progress.go`: DuelTally + ApplyDuelResult/ApplyDuelResultToQuests (metric increment, win-gating, no-attacking)
- [x] duel.go: evaluateQuestConstraints at duel start (constructed deck), warnings appended to dice banner
- [x] duel.go: applyQuestProgress at GameOver (once), engine→domain tally translation in screens/duel/quest.go
- [x] Tests quest_progress_test.go — pass; module builds
- Note: gold-card double-count caveat documented for 2-color cast quests (rare, favors player).

## Phase 5 — Wiseman offer/reward ✅ DONE
- [x] domain/quest_reward.go: QuestDef.GenerateQuest(progression) + tier-scaled rewards + RedeemDeckQuest
- [x] wiseman.go: maybeOfferDeckQuest (50% when slot free), pickDeckQuestDef (no dupes), questProgressionLevel
- [x] acceptQuest routes deck quests to AddDeckQuest; offer text for action/constraint + "Edit your deck" hint
- [x] Tests quest_reward_test.go — pass
- Note: deck-quest reward redeemed at town entry (Phase 6) via RedeemDeckQuest, not legacy giveReward.

## Phase 6 — World-frame UI ✅ DONE
- [x] Embedded Questnew.spr.png (QuestScroll_png); 2-frame scroll drawn lower-right (empty/open by quest count)
- [x] Click scroll toggles overlay panel listing each quest: title, description, progress / win-status, days left
- [x] RedeemFulfilledDeckQuests on entering ANY town (level.go) + reward SFX; ExpireDeckQuests on day tick
- [x] Wiseman-test isolation fixed (fullDeckQuestSlots helper) for legacy boon paths

## Phase 7 — Persistence + verification ✅ DONE
- [x] Save round-trips new Quest fields (JSON via Level.Player) — TestDeckQuestJSONRoundTrip
- [x] go test ./... green (s30) + full mage-go suite green
- [x] golangci-lint: s30 clean (0 issues); mage-go changed packages clean (84 issues are pre-existing in interactive/ai/*)

## Phase 8 — Quest reward screen ✅ DONE
- [x] `domain.RedeemDeckQuest` returns granted cards; `RedeemFulfilledDeckQuests` returns `[]DeckQuestReward` (quest+gold+cards)
- [x] New `QuestRewardScr` screen name + `screens/quest_reward.go` overlay: dimmed backdrop + Winbk_Questn.pic.png panel, shows each quest's title, +gold, and won card images
- [x] level.go shows reward screen on town entry when quests redeemed, then enters town via `enterCityScreen` helper on dismiss
- [x] Tests: TestRedeemFulfilledDeckQuests + updated RedeemDeckQuest; full suite green, lint clean

## Phase 9 — Unify quest storage ✅ DONE
- [x] Removed legacy `Player.ActiveQuest *Quest`; everything now in `ActiveQuests []*Quest` (cap `MaxActiveQuests`=3, legacy + deck share slots)
- [x] Added `LegacyQuest()`/`HasLegacyQuest()`/`RemoveQuest(q)`; renamed `CanAcceptQuest`/`AddQuest`/`HasQuest`
- [x] `ExpireDeckQuests`/`RedeemFulfilledDeckQuests` guard `IsDeckChanging()` so legacy quests still expire via Wiseman (city ban) and redeem at their city
- [x] Updated wiseman.go (LegacyQuest lookups, giveReward removes its quest, capacity gate), city.go, minimap.go, duel.go
- [x] `deckQuestOfferChance` is now a test-settable var; tests use `disableDeckQuestOffers(t)` instead of filling slots
- [x] All tests pass (×10, no flakiness); lint clean

## Phase 10 — Simplify reward model + merge colors ✅ DONE
- [x] One `QuestReward{Gold,Cards,CardColor,Amulets,AmuletColor,ManaLinks}` on every quest; granted via `GrantQuestReward`
- [x] Removed `RewardType` enum + parallel reward fields (RewardGold/RewardCardColor/RewardCardCount/RewardAmulets/AmuletColor/RewardCard)
- [x] Legacy delivery/defeat → `RandomSingleReward` (1 of a single kind); deck quests → tier bundles (`QuestDef.buildReward`)
- [x] Merged `MetricColor`+`ConstraintColor` → single `Color`; `MetricCardTypes` → `CardTypes`
- [x] `QuestReward.Description()` replaces wiseman's rewardDescription/deckQuestRewardDescription; reward SFX picked from contents
- [x] All tests updated + pass (×10); lint clean

## Remaining / nice-to-have (not blocking)
- Scroll placement consts (questScrollX/Y) are best-guess; verify visually in-app and nudge if overlapping frame art.
- Constraint-quest deck warning shows in duel dice banner; could also surface in the pre-duel ante screen.
- Manual playtest: accept each quest type, watch progress, rebuild for a constraint, redeem in town.
