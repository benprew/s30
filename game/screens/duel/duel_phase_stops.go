package duel

import (
	"github.com/benprew/mage-go/pkg/mage/core"
	"image"
)

var phaseSteps = [phaseCount][]core.PhaseStep{
	{core.Untap}, {core.Upkeep}, {core.Draw}, {core.PrecombatMain},
	{core.BeginCombat, core.DeclareAttackers, core.DeclareBlockers, core.FirstStrikeDamage, core.CombatDamage, core.EndCombat},
	{core.PostcombatMain}, {core.Cleanup}, {core.EndStep},
}

func phaseButtonBounds(idx int, own bool) image.Rectangle {
	pos := phasePOS(idx, own)
	return image.Rect(253, pos.Y+4, 288, pos.Y+44)
}

func (s *DuelScreen) handlePhaseClick(x, y int) bool {
	if s.human == nil {
		return false
	}
	for _, own := range []bool{false, true} {
		for idx, steps := range phaseSteps {
			if !image.Pt(x, y).In(phaseButtonBounds(idx, own)) {
				continue
			}
			stop := !s.human.PhaseStop(steps[0], own)
			for _, step := range steps {
				s.human.SetPhaseStop(step, own, stop)
			}
			return true
		}
	}
	return false
}
