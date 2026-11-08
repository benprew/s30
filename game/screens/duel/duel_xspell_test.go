package duel

import (
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
)

func TestEnterXChoosingModeDefaultsToMaxX(t *testing.T) {
	s := &DuelScreen{}
	action := interactive.ActionOption{
		Type:      interactive.ActionCastSpell,
		CardID:    uuid.New(),
		CardName:  "Fireball",
		NeedsX:    true,
		MaxXValue: 14,
	}

	s.enterXChoosingMode([]interactive.ActionOption{action})

	if !s.isChoosingX() {
		t.Fatal("expected isChoosingX() to be true")
	}
	if s.xMaxValue != 14 {
		t.Fatalf("xMaxValue = %d, want 14", s.xMaxValue)
	}
	if s.xCurrentValue != 14 {
		t.Fatalf("xCurrentValue = %d, want 14 (default to MaxXValue)", s.xCurrentValue)
	}
}

func TestXChooserLayoutFitsWithinScreenForLargeX(t *testing.T) {
	s := &DuelScreen{}
	action := interactive.ActionOption{
		Type:      interactive.ActionCastSpell,
		CardID:    uuid.New(),
		CardName:  "Fireball",
		NeedsX:    true,
		MaxXValue: 25,
	}
	s.enterXChoosingMode([]interactive.ActionOption{action})

	layout := s.xChooserLayout(1024, 768, 340)
	if layout.StartY < 0 {
		t.Errorf("StartY = %d, overflows top of screen (< 0)", layout.StartY)
	}
	if layout.BottomY > 768 {
		t.Errorf("BottomY = %d, overflows bottom of 768px screen", layout.BottomY)
	}
	if layout.Minus.Empty() || layout.Plus.Empty() || layout.Confirm.Empty() {
		t.Fatalf("expected non-empty Minus, Plus, and Confirm bounds: %+v", layout)
	}
	if layout.Minus.Max.X > layout.Value.Min.X || layout.Value.Max.X > layout.Plus.Min.X {
		t.Errorf("expected Minus, Value, Plus ordered left-to-right without overlap: %+v", layout)
	}
	if layout.Confirm.Min.Y < layout.Minus.Max.Y {
		t.Errorf("expected Confirm button below selector row: %+v", layout)
	}
}

func TestXChooserMouseAdjustsAndConfirmsValue(t *testing.T) {
	human, responses := newTestHuman()
	action := interactive.ActionOption{
		Type:      interactive.ActionCastSpell,
		CardID:    uuid.New(),
		CardName:  "Hurricane",
		NeedsX:    true,
		MaxXValue: 12,
	}
	s := &DuelScreen{human: human}
	s.enterXChoosingMode([]interactive.ActionOption{action})

	layout := s.xChooserLayout(1024, 768, 0)
	minusPt := layout.Minus.Min.Add(layout.Minus.Size().Div(2))
	plusPt := layout.Plus.Min.Add(layout.Plus.Size().Div(2))
	confirmPt := layout.Confirm.Min.Add(layout.Confirm.Size().Div(2))

	s.updateXChoosingMouse(plusPt.X, plusPt.Y, true, 1024, 768)
	if s.xCurrentValue != 12 {
		t.Fatalf("after plus at max, xCurrentValue = %d, want 12", s.xCurrentValue)
	}

	s.updateXChoosingMouse(minusPt.X, minusPt.Y, true, 1024, 768)
	s.updateXChoosingMouse(minusPt.X, minusPt.Y, true, 1024, 768)
	if s.xCurrentValue != 10 {
		t.Fatalf("after two minus clicks, xCurrentValue = %d, want 10", s.xCurrentValue)
	}

	s.updateXChoosingMouse(plusPt.X, plusPt.Y, true, 1024, 768)
	if s.xCurrentValue != 11 {
		t.Fatalf("after plus click, xCurrentValue = %d, want 11", s.xCurrentValue)
	}

	s.updateXChoosingMouse(confirmPt.X, confirmPt.Y, true, 1024, 768)
	if s.isChoosingX() {
		t.Fatal("expected isChoosingX() to be false after confirming")
	}

	select {
	case got := <-responses:
		if got.XValue != 11 {
			t.Fatalf("submitted XValue = %d, want 11", got.XValue)
		}
	default:
		t.Fatal("expected priority action to be sent on Confirm")
	}
}

func TestXChooserClampsAtZero(t *testing.T) {
	s := &DuelScreen{}
	action := interactive.ActionOption{
		Type:      interactive.ActionCastSpell,
		CardID:    uuid.New(),
		CardName:  "Fireball",
		NeedsX:    true,
		MaxXValue: 1,
	}
	s.enterXChoosingMode([]interactive.ActionOption{action})

	s.decrementX()
	if s.xCurrentValue != 0 {
		t.Fatalf("xCurrentValue = %d, want 0", s.xCurrentValue)
	}
	s.decrementX()
	if s.xCurrentValue != 0 {
		t.Fatalf("xCurrentValue = %d, want 0 (clamped)", s.xCurrentValue)
	}
}

func TestXChooserDigitInputSupportsMultiDigitX(t *testing.T) {
	s := &DuelScreen{}
	action := interactive.ActionOption{
		Type:      interactive.ActionCastSpell,
		CardID:    uuid.New(),
		CardName:  "Fireball",
		NeedsX:    true,
		MaxXValue: 15,
	}
	s.enterXChoosingMode([]interactive.ActionOption{action})

	s.typeXDigit(1)
	if s.xCurrentValue != 1 {
		t.Fatalf("after typing 1, xCurrentValue = %d, want 1", s.xCurrentValue)
	}
	s.typeXDigit(2)
	if s.xCurrentValue != 12 {
		t.Fatalf("after typing 1 then 2, xCurrentValue = %d, want 12", s.xCurrentValue)
	}
	s.typeXDigit(3)
	if s.xCurrentValue != 3 {
		t.Fatalf("after typing 3 (123 > 15), xCurrentValue = %d, want 3", s.xCurrentValue)
	}
}
