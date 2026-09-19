package duel

import (
	"image"
	"strings"
	"testing"

	"github.com/google/uuid"

	mage "github.com/benprew/mage-go/pkg/mage"
	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai"
	"github.com/benprew/mage-go/pkg/mage/interactive/ai/heuristic"
)

func TestRectIndexAtPoint(t *testing.T) {
	rects := []image.Rectangle{
		image.Rect(10, 20, 30, 50),
		image.Rect(40, 20, 60, 50),
	}

	tests := []struct {
		name string
		x    int
		y    int
		want int
	}{
		{name: "first card", x: 10, y: 20, want: 0},
		{name: "second card", x: 59, y: 49, want: 1},
		{name: "right edge excluded", x: 60, y: 49, want: -1},
		{name: "gap", x: 35, y: 30, want: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rectIndexAtPoint(rects, tt.x, tt.y); got != tt.want {
				t.Fatalf("rectIndexAtPoint() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMulliganPreviewPositionUsesOppositeSide(t *testing.T) {
	const screenW, screenH = 1024, 768
	previewBounds := image.Rect(0, 0, 245, 342)

	leftCard := image.Rect(50, 300, 170, 468)
	if got := mulliganPreviewPosition(screenW, screenH, leftCard, previewBounds); got.X != 759 {
		t.Fatalf("preview for left card X = %d, want 759", got.X)
	}

	rightCard := image.Rect(850, 300, 970, 468)
	if got := mulliganPreviewPosition(screenW, screenH, rightCard, previewBounds); got.X != 20 {
		t.Fatalf("preview for right card X = %d, want 20", got.X)
	}
}

func TestMulliganPreviewPositionStaysOnScreen(t *testing.T) {
	card := image.Rect(5, 40, 25, 68)
	previewBounds := image.Rect(0, 0, 245, 342)

	got := mulliganPreviewPosition(200, 180, card, previewBounds)
	if got.X != 0 || got.Y != 0 {
		t.Fatalf("preview position = %v, want (0,0)", got)
	}
}

func TestConcedeControlsUseMulliganButtonImages(t *testing.T) {
	s := &DuelScreen{}
	s.initMulligan()

	if s.cancelBtn == nil || s.concedeBtn == nil || s.concedeConfirmBtn == nil || s.concedeKeepBtn == nil {
		t.Fatal("concede controls should use standard buttons")
	}
	if s.cancelBtn.Normal != s.mulliganKeepBtn.Normal ||
		s.concedeBtn.Normal != s.mulliganKeepBtn.Normal ||
		s.concedeConfirmBtn.Hover != s.mulliganKeepBtn.Hover ||
		s.concedeKeepBtn.Pressed != s.mulliganKeepBtn.Pressed {
		t.Fatal("concede controls should share the mulligan button images")
	}
	if s.concedeBtn.ButtonText.Text != "Concede" ||
		s.concedeConfirmBtn.ButtonText.Text != "Concede" ||
		s.concedeKeepBtn.ButtonText.Text != "Keep Playing" {
		t.Fatal("concede controls should have the expected labels")
	}
}

// newMulliganScreen builds a duel screen that is sitting on the mulligan
// prompt with a freshly dealt seven-card hand. loopCancel is pre-set so
// finishMulligan does not start a game loop for the nil game.
func newMulliganScreen(t *testing.T) (*DuelScreen, *interactive.HumanPlayer) {
	t.Helper()
	human := interactive.NewHumanPlayer("You")
	aiPlayer := ai.NewAIPlayer("Opponent", heuristic.NewAdaptive())
	for _, p := range []mage.Player{human, aiPlayer} {
		for range 40 {
			c, err := mage.CreateCard("Forest")
			if err != nil {
				t.Fatalf("CreateCard: %v", err)
			}
			c.SetOwner(p.PlayerID())
			p.AddToLibrary(c)
		}
		for range 7 {
			p.DrawCard()
		}
	}
	s := &DuelScreen{human: human, aiPlayer: aiPlayer, loopCancel: func() {}}
	s.initMulligan()
	return s, human
}

func TestMulliganKeepPutsCardsOnTheBottom(t *testing.T) {
	s, human := newMulliganScreen(t)

	s.doHumanMulligan()
	s.doHumanMulligan()
	if got := len(human.Hand()); got != 7 {
		t.Fatalf("hand after two mulligans = %d, want 7 (London rules redeal seven)", got)
	}

	bottomed := make(map[uuid.UUID]bool)
	for _, c := range human.Hand()[:s.mulliganCount] {
		s.mulliganSelected[c.ID()] = true
		bottomed[c.ID()] = true
	}
	lib := len(human.Library())
	s.finishMulligan()

	if got := len(human.Hand()); got != 5 {
		t.Fatalf("hand after keeping on two mulligans = %d, want 5", got)
	}
	if got := len(human.Library()); got != lib+2 {
		t.Fatalf("library after keeping = %d, want %d", got, lib+2)
	}
	for _, c := range human.Library()[len(human.Library())-2:] {
		if !bottomed[c.ID()] {
			t.Fatal("selected cards should sit at the bottom of the library")
		}
	}
}

func TestSeventhMulliganEmptiesTheHand(t *testing.T) {
	s, human := newMulliganScreen(t)

	for range maxMulligans {
		s.doHumanMulligan()
	}

	if s.inMulligan {
		t.Fatal("the mulligan loop should end after the seventh mulligan")
	}
	if got := len(human.Hand()); got != 0 {
		t.Fatalf("hand after seven mulligans = %d, want 0 (seven cards go to the bottom)", got)
	}
	if got := len(human.Library()); got != 40 {
		t.Fatalf("library after seven mulligans = %d, want 40", got)
	}
}

func TestMulliganTitleNamesTheCostOfKeeping(t *testing.T) {
	if got := mulliganTitle(0, 0, false); got != "Opening hand — Keep or Mulligan?" {
		t.Fatalf("title for the first hand = %q", got)
	}
	if got := mulliganTitle(2, 0, false); !strings.Contains(got, "2 card") {
		t.Fatalf("title after two mulligans = %q, want the number of cards keeping costs", got)
	}
	if got := mulliganTitle(2, 1, true); !strings.Contains(got, "1 selected") {
		t.Fatalf("bottoming title = %q, want the selection count", got)
	}
}

func TestMulliganMiniCardsFitWithoutOverlap(t *testing.T) {
	s, _ := newMulliganScreen(t)
	rects := s.mulliganCardRects(1024, 768)
	if len(rects) != 7 {
		t.Fatalf("card count = %d", len(rects))
	}
	area := image.Rect(0, 90, 1024, 688)
	for i, rect := range rects {
		if rect.Size() != image.Pt(183, 256) || !rect.In(area) {
			t.Fatalf("card %d bounds = %v", i, rect)
		}
		for _, other := range rects[i+1:] {
			if rect.Overlaps(other) {
				t.Fatal("mulligan cards overlap")
			}
		}
	}
}
