---
name: s30-player-changelog
description: Summarize changes between two Git revisions in the S30 game and mage-go rules-engine repositories as a player-facing changelog. Use for release notes, version comparisons, or requests about what changed in both repositories; do not use for implementation work or a technical code review.
---

# S30 Player Changelog

Create a concise, evidence-based list of the changes that matter most to people who play S30. Treat S30 and mage-go as one product: S30 contains the game and user interface, while mage-go supplies rules, cards, and AI behavior.

## Resolve the comparison

- Accept one start/end ref pair for both repositories or separate pairs when the user supplies them.
- Locate both checkouts. They are normally `$HOME/src/s30` and `$HOME/src/mage-go`; if those paths are unavailable, find the current S30 checkout and its sibling `mage-go` checkout.
- Verify every ref in its repository before analysis. Do not silently substitute a similarly named branch or tag.
- Compare committed trees with `git diff <start>..<end>`. Exclude untracked and working-tree changes unless the user explicitly asks to include them.
- Check ancestry with `git merge-base --is-ancestor`. If history was rewritten or the start ref is not an ancestor, say so briefly and treat the direct tree diff as authoritative. Do not describe a revision-list count as “commits since” in that case.

Use the log, diff statistics, changed-file list, focused source diffs, and relevant tests as evidence. Commit titles are leads, not proof: inspect the implementation when a title is vague, overly technical, or potentially misleading.

## Select player-relevant changes

Prioritize changes in this order:

1. New game features and newly supported platforms or ways to play.
2. Save persistence, save compatibility, crash recovery, and fixes that prevent lost progress.
3. Gameplay and rules fixes that change duel outcomes.
4. New cards, sets, encounters, rewards, economy, balance, deck construction, and AI behavior.
5. Noticeable user-interface, audio, loading-time, frame-rate, memory, and input improvements.
6. Developer or build changes only when they affect installation, supported systems, or the ability to run the game.

Usually omit internal refactors, test-only work, lint changes, cache configuration, CI mechanics, documentation edits, profiling helpers, and raw line counts. Mention them only when they explain a player-visible improvement or a material release risk.

Give extra weight to platform and persistence changes. For example, adding saves to WASM or mobile builds is a headline feature, even if the diff is smaller than an engine refactor.

## Translate implementation into player impact

- Describe what a player can now do, what behaves differently, or which failure no longer occurs.
- Prefer “Web and Android games now retain saves” over file names or storage implementation details.
- Prefer “Mana payment no longer partially spends resources when a cast fails” over descriptions of transaction types.
- Name affected cards or mechanics when they make a rules fix understandable.
- Distinguish confirmed behavior from an inference. Do not claim a player-visible result solely because an internal API exists.
- Combine related S30 and mage-go work into one change when they implement the same outcome. Avoid counting an engine feature and its S30 integration twice.
- Call out compatibility requirements between the repositories when the compared S30 revision needs a mage-go revision that its `go.mod` does not select directly.

## Write the changelog

Lead with the most important changes. Use short bullets in descending order of player impact. Group bullets under plain-language headings such as “New features,” “Gameplay and rules fixes,” “Cards and balance,” and “Platform and reliability” only when grouping makes the result easier to scan.

Include the exact refs and abbreviated commit IDs for both repositories. Add a short comparison caveat when histories diverge or untracked files were excluded. Keep technical implementation details out of the main list; put essential repository or dependency notes in a brief final note.

Do not edit either repository while preparing the summary. Do not run full test suites unless the user also asks for validation.
