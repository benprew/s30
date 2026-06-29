package screens

import (
	"fmt"
	"image/color"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Quest overlay panel placement (in 1024x768 design coords).
const (
	questPanelX = 150
	questPanelY = 110
	questPanelW = 724
	questPanelH = 500
)

// QuestScrollScreen is the transparent overlay listing the player's active
// quests, opened from the world frame's quest scroll and dismissed with a click
// or Escape. The screen underneath (world/city) is drawn beneath it.
type QuestScrollScreen struct {
	player  *domain.Player
	panelBg *ebiten.Image
}

func NewQuestScrollScreen(p *domain.Player) *QuestScrollScreen {
	panelBg := ebiten.NewImage(questPanelW, questPanelH)
	panelBg.Fill(color.RGBA{20, 12, 4, 220})
	return &QuestScrollScreen{
		player:  p,
		panelBg: panelBg,
	}
}

func (s *QuestScrollScreen) IsFramed() bool { return true }

func (s *QuestScrollScreen) IsOverlay() bool { return true }

func (s *QuestScrollScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return screenui.PopScr, nil, nil
	}
	return screenui.QuestScrollScr, nil, nil
}

func (s *QuestScrollScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Scale(scale, scale)
	panelOpts.GeoM.Translate(float64(questPanelX)*scale, float64(questPanelY)*scale)
	screen.DrawImage(s.panelBg, panelOpts)

	y := questPanelY + 16
	for _, line := range s.questPanelLines() {
		txt := elements.NewText(20, line, questPanelX+20, y)
		txt.Color = color.White
		txt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		y += 26
	}
}

func (s *QuestScrollScreen) questPanelLines() []string {
	lines := []string{"Active Quests", ""}
	if len(s.player.ActiveQuests) == 0 {
		return append(lines, "No active quests.", "", "Visit a Wiseman to take one.")
	}
	for _, q := range s.player.ActiveQuests {
		lines = append(lines, q.Title, "  "+q.Description)
		switch q.Type {
		case domain.QuestTypeActionTracker:
			lines = append(lines, fmt.Sprintf("  Progress: %d / %d", q.Progress, q.Target))
		case domain.QuestTypeDeckConstraint:
			if q.IsCompleted {
				lines = append(lines, "  Complete - redeem in any town")
			} else {
				lines = append(lines, "  Win a duel under this rule")
			}
		}
		lines = append(lines, fmt.Sprintf("  %d days left", q.DaysRemaining), "")
	}
	return lines
}
