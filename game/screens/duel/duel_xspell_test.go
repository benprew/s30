package duel

import (
	"image"
	"testing"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestEnterXChoosingModeDefaultsToMaxX(t *testing.T) {
	s := &DuelScreen{}
	s.enterXChoosingMode([]interactive.ActionOption{{MaxXValue: 14}})

	if !s.isChoosingX() || s.xMaxValue != 14 || s.xCurrentValue != 14 {
		t.Fatalf("choosing X=%v, max=%d, current=%d, want an open chooser at X=14",
			s.isChoosingX(), s.xMaxValue, s.xCurrentValue)
	}
}

func TestXChooserLayoutFitsWithinScreenForLargeX(t *testing.T) {
	s := &DuelScreen{}
	s.enterXChoosingMode([]interactive.ActionOption{{MaxXValue: 25}})

	screen := image.Rect(0, 0, 1024, 768)
	layout := s.xChooserLayout(screen.Dx(), screen.Dy(), 340)
	for _, bounds := range []image.Rectangle{layout.minus, layout.value, layout.plus, layout.confirm} {
		if bounds.Empty() || !bounds.In(screen) {
			t.Errorf("control bounds %v must fit inside %v", bounds, screen)
		}
	}
	if layout.titleY < 0 || layout.confirm.Min.Y < layout.minus.Max.Y {
		t.Errorf("expected the title on screen and Confirm below the selector, got %+v", layout)
	}
	if layout.minus.Max.X > layout.value.Min.X || layout.value.Max.X > layout.plus.Min.X {
		t.Errorf("expected minus, value, and plus in order without overlap, got %+v", layout)
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
	click := func(bounds image.Rectangle) {
		point := bounds.Min.Add(bounds.Size().Div(2))
		s.handleXChooserClick(layout, point.In)
	}

	for _, step := range []struct {
		button image.Rectangle
		want   int
	}{
		{layout.plus, 12},
		{layout.minus, 11},
		{layout.minus, 10},
		{layout.plus, 11},
	} {
		click(step.button)
		if s.xCurrentValue != step.want {
			t.Fatalf("X after clicking %v = %d, want %d", step.button, s.xCurrentValue, step.want)
		}
	}
	select {
	case got := <-responses:
		t.Fatalf("cast submitted before confirmation: %+v", got)
	default:
	}

	click(layout.confirm)
	if s.isChoosingX() {
		t.Fatal("Confirm must close the X chooser")
	}
	select {
	case got := <-responses:
		if got.CardID != action.CardID || got.Type != action.Type || got.XValue != 11 {
			t.Fatalf("unexpected cast action: %+v", got)
		}
	default:
		t.Fatal("Confirm must submit the cast action")
	}
}

func TestXChooserClampsAtZero(t *testing.T) {
	s := &DuelScreen{}
	s.enterXChoosingMode([]interactive.ActionOption{{MaxXValue: 1}})

	for range 2 {
		s.adjustX(-1)
		if s.xCurrentValue != 0 {
			t.Fatalf("X = %d, want 0", s.xCurrentValue)
		}
	}
}

func TestXChooserDigitInputSupportsMultiDigitX(t *testing.T) {
	s := &DuelScreen{}
	s.enterXChoosingMode([]interactive.ActionOption{{MaxXValue: 15}})

	for _, step := range []struct{ digit, want int }{{1, 1}, {2, 12}, {3, 3}} {
		s.typeXDigit(step.digit)
		if s.xCurrentValue != step.want {
			t.Fatalf("X after typing %d = %d, want %d", step.digit, s.xCurrentValue, step.want)
		}
	}
}

func TestXChooserKeyboardAndWheelShortcuts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   ebiten.Key
		wheel float64
		want  int
	}{
		{name: "plus", key: ebiten.KeyEqual, want: 7},
		{name: "minus", key: ebiten.KeyMinus, want: 5},
		{name: "numpad plus", key: ebiten.KeyKPAdd, want: 7},
		{name: "numpad minus", key: ebiten.KeyKPSubtract, want: 5},
		{name: "scroll up", wheel: 0.5, want: 7},
		{name: "scroll down", wheel: -0.5, want: 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &DuelScreen{xMaxValue: 12, xCurrentValue: 6}
			s.handleXChooserShortcuts(func(key ebiten.Key) bool {
				return tc.wheel == 0 && key == tc.key
			}, tc.wheel)
			if s.xCurrentValue != tc.want {
				t.Fatalf("X = %d, want %d", s.xCurrentValue, tc.want)
			}
		})
	}
}
