package screens

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/benprew/s30/assets"
	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// This is the frame that you see when you're walking around the world and in cities
//
// World frame shows character stats, current quest, available money, etc
// And has buttons to go to the minimap
// Not technically a screen, but it has draw and update functions, so I'm including it here

// Buybuttons.spr.png - buy buttons
//
// Worlds.spr.png - sprite for world frame
// Compnew.spr.new - compass sprite in world frame
// Questnew.spr.png - quest sprite in world frame
// Clocknew.spr.png - clock sprite
// Daysnew.spr.png - 0-5 for clock
// Days.spr.png - 0-5 for clock
// Prdfrma.pic.png - food/gold/life/cards quest icons - old?
// Statbut1.pic.png - stat buttons - old?

const (
	FrameOffsetX = 100
	FrameOffsetY = 75
	FrameWidth   = 820
	FrameHeight  = 425
)

// Quest scroll placement (in 1024x768 design coords) and the click-to-open
// quest overlay panel.
const (
	questScrollX        = 815
	questScrollY        = 538
	questScrollScale    = 0.8
	questScrollFontSize = 14
	// Fraction of the scroll's width the title may occupy; the rest is the
	// curled parchment ends that text shouldn't bleed onto.
	questScrollTextFrac = 0.52
	// Downward nudge (design px) so the title centers on the flat parchment
	// rather than the full sprite, whose bottom roll is thicker than the top.
	questScrollTextDY = 4
	questPanelX      = 150
	questPanelY      = 110
	questPanelW      = 724
	questPanelH      = 500
)

type WorldFrame struct {
	Buttons       []*elements.Button
	Text          []*elements.Text
	img           *ebiten.Image
	player        *domain.Player // handle to player so we can get player stats
	amuletSprites []*ebiten.Image

	questScrollEmpty  *ebiten.Image
	questScrollActive *ebiten.Image
	questPanelBg      *ebiten.Image
	questPanelOpen    bool
}

func NewWorldFrame(p *domain.Player) (*WorldFrame, error) {
	img, err := imageutil.LoadImage(assets.WorldFrame_png)
	if err != nil {
		return nil, err
	}
	worldSprs, err := imageutil.LoadSpriteSheet(12, 5, assets.WorldSpr_png)
	if err != nil {
		return nil, err
	}

	amuletSprs, err := imageutil.LoadSpriteSheet(5, 1, assets.Amsprite_png)
	if err != nil {
		return nil, err
	}

	questSprs, err := imageutil.LoadSpriteSheet(2, 1, assets.QuestScroll_png)
	if err != nil {
		return nil, err
	}

	panelBg := ebiten.NewImage(questPanelW, questPanelH)
	panelBg.Fill(color.RGBA{20, 12, 4, 220})

	return &WorldFrame{
		img:               img,
		Buttons:           mkWfButtons(worldSprs),
		player:            p,
		amuletSprites:     amuletSprs[0],
		questScrollEmpty:  questSprs[0][0],
		questScrollActive: questSprs[0][1],
		questPanelBg:      panelBg,
	}, nil
}

func (f *WorldFrame) Draw(screen *ebiten.Image, scale float64) {
	frameOpts := &ebiten.DrawImageOptions{}
	frameOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(f.img, frameOpts)

	for _, b := range f.Buttons {
		b.Draw(screen, frameOpts, scale)
	}

	for _, t := range f.Text {
		t.Draw(screen, frameOpts, scale)
	}

	amuletPositions := []int{125, 250, 375, 500, 625}
	amuletY := 628
	for i, sprite := range f.amuletSprites {
		if i < len(amuletPositions) {
			amuletOpts := &ebiten.DrawImageOptions{}
			amuletOpts.GeoM.Scale(scale, scale)
			amuletOpts.GeoM.Translate(float64(amuletPositions[i])*scale, float64(amuletY)*scale)
			screen.DrawImage(sprite, amuletOpts)
		}
	}

	f.drawQuestScroll(screen, scale)
}

// drawQuestScroll draws the lower-right quest scroll (open when the player has
// active quests) and, if toggled open, the quest overlay panel.
func (f *WorldFrame) drawQuestScroll(screen *ebiten.Image, scale float64) {
	frame := f.questScrollEmpty
	if len(f.player.ActiveQuests) > 0 {
		frame = f.questScrollActive
	}
	if frame != nil {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(questScrollScale, questScrollScale)
		opts.GeoM.Scale(scale, scale)
		opts.GeoM.Translate(float64(questScrollX)*scale, float64(questScrollY)*scale)
		screen.DrawImage(frame, opts)

		if len(f.player.ActiveQuests) > 0 {
			f.drawQuestScrollTitle(screen, frame, scale)
		}
	}

	if !f.questPanelOpen {
		return
	}

	panelOpts := &ebiten.DrawImageOptions{}
	panelOpts.GeoM.Scale(scale, scale)
	panelOpts.GeoM.Translate(float64(questPanelX)*scale, float64(questPanelY)*scale)
	screen.DrawImage(f.questPanelBg, panelOpts)

	y := questPanelY + 16
	for _, line := range f.questPanelLines() {
		txt := elements.NewText(20, line, questPanelX+20, y)
		txt.Color = color.White
		txt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
		y += 26
	}
}

// drawQuestScrollTitle prints the active quest's name, suffixed with an
// ellipsis, centered on the parchment so it reads as a clickable summary.
func (f *WorldFrame) drawQuestScrollTitle(screen, frame *ebiten.Image, scale float64) {
	scrollW := float64(frame.Bounds().Dx()) * questScrollScale
	scrollH := float64(frame.Bounds().Dy()) * questScrollScale

	label := fitQuestTitle(f.player.ActiveQuests[0].Title, scrollW*questScrollTextFrac)
	txt := elements.NewText(questScrollFontSize, label, questScrollX, questScrollY+questScrollTextDY)
	txt.Color = color.White
	txt.BoundsW = scrollW
	txt.BoundsH = scrollH
	txt.HAlign = elements.AlignCenter
	txt.VAlign = elements.AlignMiddle
	txt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
}

// fitQuestTitle returns the quest title plus a trailing ellipsis, dropping
// trailing characters until "title…" fits within maxW design-space pixels.
func fitQuestTitle(title string, maxW float64) string {
	measure := func(s string) float64 {
		w, _ := elements.NewText(questScrollFontSize, s, 0, 0).Measure()
		return w
	}
	runes := []rune(title)
	for {
		label := strings.TrimRight(string(runes), " ") + "..."
		if len(runes) == 0 || measure(label) <= maxW {
			return label
		}
		runes = runes[:len(runes)-1]
	}
}

func (f *WorldFrame) questPanelLines() []string {
	lines := []string{"Active Quests", ""}
	if len(f.player.ActiveQuests) == 0 {
		return append(lines, "No active quests.", "", "Visit a Wiseman to take one.")
	}
	for _, q := range f.player.ActiveQuests {
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

func (f *WorldFrame) questScrollBounds(scale float64) image.Rectangle {
	if f.questScrollEmpty == nil {
		return image.Rectangle{}
	}
	w := float64(f.questScrollEmpty.Bounds().Dx()) * questScrollScale * scale
	h := float64(f.questScrollEmpty.Bounds().Dy()) * questScrollScale * scale
	x := float64(questScrollX) * scale
	y := float64(questScrollY) * scale
	return image.Rect(int(x), int(y), int(x+w), int(y+h))
}

func (f *WorldFrame) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	options := &ebiten.DrawImageOptions{}
	for i := range f.Buttons {
		b := f.Buttons[i]
		b.Update(options, scale, W, H)
		if b.ID == "minimap" && b.IsClicked() {
			return screenui.MiniMapScr, nil, nil
		}
		if (b.ID == "character" || b.ID == "book") && b.IsClicked() {
			if am := gameaudio.Get(); am != nil {
				am.PlaySFX(gameaudio.SFXStatsScreen)
			}
		}
	}

	f.updateQuestScroll(scale)

	f.Text = mkWfText(f.player)

	return -1, nil, nil
}

// updateQuestScroll toggles the quest overlay when the scroll is clicked, and
// closes it on a click elsewhere.
func (f *WorldFrame) updateQuestScroll(scale float64) {
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ui.TouchPosition()
	if mx == 0 {
		mx, my = ebiten.CursorPosition()
	}
	if (image.Point{X: mx, Y: my}).In(f.questScrollBounds(scale)) {
		f.questPanelOpen = !f.questPanelOpen
		if am := gameaudio.Get(); am != nil {
			am.PlaySFX(gameaudio.SFXClick)
		}
		return
	}
	f.questPanelOpen = false
}

func mkWfButtons(worldSprs [][]*ebiten.Image) []*elements.Button {
	sidebar := []string{"book", "minimap", "dungeon", "character"}
	buttons := []*elements.Button{}
	for i, n := range sidebar {
		offset := i * 90
		y := 110 + offset
		normalImg := worldSprs[4][i*2]
		btn := elements.NewButton(normalImg, worldSprs[4][i*2+1], normalImg, 8, y, 1.7)
		btn.ID = n
		buttons = append(buttons, btn)
	}
	return buttons
}

func mkWfText(p *domain.Player) []*elements.Text {
	amuletCounts := p.GetAmuletCount()

	texts := []*elements.Text{
		elements.NewText(30, fmt.Sprintf("%d", p.Gold), 140, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.Food), 270, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.Life), 400, 560),
		elements.NewText(30, fmt.Sprintf("%d", p.NumCards()), 530, 560),
	}

	amuletColors := []domain.ColorMask{
		domain.ColorWhite,
		domain.ColorBlue,
		domain.ColorBlack,
		domain.ColorRed,
		domain.ColorGreen,
	}
	amuletPositions := []int{125, 250, 375, 500, 625}
	amuletY := 648

	for i, color := range amuletColors {
		if i < len(amuletPositions) {
			count := amuletCounts[color]
			x := amuletPositions[i] + 20
			texts = append(texts, elements.NewText(18, fmt.Sprintf("%d", count), x, amuletY))
		}
	}

	return texts
}
