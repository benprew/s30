package screens

import "git.sr.ht/~cdcarter/mage-go/pkg/mage/interactive"

// SCAFFOLD: this file exists only because the branch is mid-refactor — the
// real duel_ability_chooser.go was removed and its replacement is incomplete,
// leaving game/screens uncompilable. These no-op stubs unblock builds for
// unrelated work (the dungeon system) and should be deleted once the ability
// chooser is restored or the references in duel.go are cleaned up.

func (s *DuelScreen) isChoosingAbility() bool                          { return false }
func (s *DuelScreen) enterAbilityChoosingMode(_ []interactive.ActionOption) {}
func (s *DuelScreen) exitAbilityChoosingMode()                         {}
func (s *DuelScreen) updateAbilityChoosingUI()                         {}
func (s *DuelScreen) drawAbilityChoosingUI(_, _, _ any)                {}
func (s *DuelScreen) selectAbility(_ int)                              {}

// maxX returns the X value cap for an action option. The mage-go engine does
// not (yet) expose this on ActionOption; until it does, the stub returns 0,
// which disables X-cost prompts.
func maxX(_ interactive.ActionOption) int { return 0 }
