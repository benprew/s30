# Changes: March 7 - April 21, 2026

73 commits, 601 files changed, ~11k insertions, ~12k deletions.

## Engine Migration
- **Migrated to mage-go** as the MTG rules engine, removing the entire in-house `mtg` package and subpackages. This was the biggest change — blockers, card abilities, and card choices are now handled by mage-go.
- Limited card sets while building out card rules in mage-go.

## Android Support
- Added **Android build** pipeline with GitHub Actions workflow, ebitenmobile integration, app signing, loading screen, touch input fixes, and TPS configuration.

## UI & Gameplay
- **Start screen** added with title/news screen images.
- **Duel screen improvements**: card choice UI (e.g. Healing Salve), combat indicator boxes (yellow/green instead of `*`), attachment drawing, battlefield card positioning, AI blocking visualization, and blocking logic from mage-go.
- **Wiseman boons** expanded — boons can now be given at other cities if the player has an active quest.
- **Buttons reworked** across duel screen and wiseman.
- Winning a duel now gives the enemy's ante card (was random before).
- Vintage cards excluded from shops and starting decks.
- Starting deck generation uses **card power tiers** instead of rarity.
- Card prices normalized to 5-150 range.
- City card generation moved to city creation time.

## Performance & Debugging
- **Sprite registry cache** to avoid redundant image decoding.
- **Text rendering cache**.
- Audio context reuse to avoid re-initialization.
- Removed image scaling from `Draw()` functions.
- Added **pprof server** for live memory debugging.

## Audio
- Added **sound effects** with OGG format audio files.

## Card Power Scoring
- Added a **heuristic card power scorer** based on card evaluation frameworks.
- Added a **BERT model** for card power ranking.
- Updated card tiers after manual review.

## Tooling & Docs
- Python linting/venv setup with ruff and ty.
- README, CONTRIBUTING guide, and GPLv2 LICENSE added.
- Architecture doc (4+1 model) and duel screen design doc.
- Card serialization made serializable with tests.
