package duel

import (
	"image"
	"testing"
	"time"
	"uuid"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestBattlefieldLandGroups(t *testing.T) {
	for _, tc := range []struct {
		name      string
		groupSize int
	}{
		{name: "mixed", groupSize: 2},
		{name: "crowded", groupSize: 12},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &DuelScreen{
				self:          &duelPlayer{name: "player", handX: duelHandX},
				opponent:      &duelPlayer{name: "opponent", handX: duelHandX},
				lastMsg:       &interactive.GameMsg{State: &interactive.GameState{}},
				cardPositions: make(map[uuid.UUID]image.Point),
			}
			screen := ebiten.NewImage(1024, 768)
			defer screen.Deallocate()
			for _, dp := range []*duelPlayer{s.self, s.opponent} {
				ps := s.playerState(dp)
				for range tc.groupSize {
					ps.Battlefield = append(ps.Battlefield,
						interactive.PermanentState{ID: uuid.New(), Name: "Forest", IsLand: true, Tapped: true},
						interactive.PermanentState{ID: uuid.New(), Name: "Mountain", IsLand: true})
				}
				s.drawBattlefield(screen, dp, ps, time.Now())

				untappedRight, tappedLeft := 0, dp.handX
				rowY := s.cardPositions[ps.Battlefield[0].ID].Y
				for _, land := range ps.Battlefield {
					pos := s.cardPositions[land.ID]
					w, _ := fieldCardSize(land.Tapped)
					if pos.Y != rowY || pos.X < duelBoardX || pos.X+w > dp.handX {
						t.Fatalf("%s land at %v is outside the land row", dp.name, pos)
					}
					if land.Tapped {
						tappedLeft = min(tappedLeft, pos.X)
					} else {
						untappedRight = max(untappedRight, pos.X+w)
					}
				}
				if gap := tappedLeft - untappedRight; gap < 16 {
					t.Fatalf("%s land groups have a %d px gap, want at least 16 px", dp.name, gap)
				}
				if hit := s.fieldPermAtPoint(tappedLeft-1, rowY+10, dp); hit != nil {
					t.Fatalf("click in %s land gap selected %s", dp.name, hit.Name)
				}
				for _, land := range ps.Battlefield {
					pos := s.cardPositions[land.ID]
					_, h := fieldCardSize(land.Tapped)
					if hit := s.fieldPermAtPoint(pos.X+1, pos.Y+h-1, dp); hit == nil || hit.ID != land.ID {
						t.Fatalf("click on %s land at %v did not select %s", dp.name, pos, land.Name)
					}
				}
			}
		})
	}
}
