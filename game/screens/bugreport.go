package screens

import (
	"fmt"
	"image/color"
	"time"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/bugreport"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	bugPanelX = 162
	bugPanelY = 110
	bugPanelW = 700
	bugPanelH = 540
)

// BugReportScreen provides an in-game modal dialog to report issues with full state capture.
type BugReportScreen struct {
	level          *world.Level
	prevScreenName screenui.ScreenName
	prevScreen     screenui.Screen
	textInput      *elements.TextInput
	buttons        []*elements.Button
	submitBtn      *elements.Button
	copyBtn        *elements.Button
	closeBtn       *elements.Button
	panelBg        *ebiten.Image
	submitter      bugreport.Submitter
	statusMsg      string
	statusColor    color.Color
	submitting     bool
	submittedAt    time.Time
}

// NewBugReportScreen creates a new BugReportScreen overlay.
func NewBugReportScreen(level *world.Level, prevScreenName screenui.ScreenName, prevScreen screenui.Screen) *BugReportScreen {
	panelBg := ebiten.NewImage(bugPanelW, bugPanelH)
	panelBg.Fill(color.RGBA{18, 14, 10, 245})

	input := elements.NewTextInput(bugPanelX+30, bugPanelY+100, bugPanelW-60, 260, "Describe what happened and any steps to reproduce...")

	s := &BugReportScreen{
		level:          level,
		prevScreenName: prevScreenName,
		prevScreen:     prevScreen,
		textInput:      input,
		panelBg:        panelBg,
		submitter:      bugreport.NewDefaultSubmitter(),
		statusColor:    color.RGBA{220, 220, 220, 255},
	}

	s.setupButtons()
	return s
}

func (s *BugReportScreen) IsFramed() bool { return false }

func (s *BugReportScreen) IsOverlay() bool { return true }

func (s *BugReportScreen) setupButtons() {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}

	btnY := bugPanelY + bugPanelH - 70
	submitW, _ := elements.TextButtonSize("Submit", fontFace)
	copyW, _ := elements.TextButtonSize("Copy & Save", fontFace)
	closeW, _ := elements.TextButtonSize("Close", fontFace)

	gap := 16
	totalW := submitW + gap + copyW + gap + closeW
	startX := bugPanelX + (bugPanelW-totalW)/2

	s.submitBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: "Submit", Font: fontFace, ID: "submit",
		X: startX, Y: btnY,
	})

	s.copyBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: "Copy & Save", Font: fontFace, ID: "copy_save",
		X: startX + submitW + gap, Y: btnY,
	})

	s.closeBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: "Close", Font: fontFace, ID: "close",
		X: startX + submitW + gap + copyW + gap, Y: btnY,
	})

	s.buttons = []*elements.Button{s.submitBtn, s.copyBtn, s.closeBtn}
}

func (s *BugReportScreen) SubmitSync() (*bugreport.SubmitResult, error) {
	s.submitting = true
	s.statusMsg = "Submitting report..."
	s.statusColor = color.RGBA{220, 200, 100, 255}

	report := bugreport.CollectReport(
		s.level,
		screenui.ScreenNameToString(s.prevScreenName),
		s.prevScreen,
		s.textInput.Text(),
	)

	result, err := s.submitter.Submit(report)
	s.submitting = false
	s.submittedAt = time.Now()

	if err != nil {
		s.statusMsg = fmt.Sprintf("Submission failed: %v", err)
		s.statusColor = color.RGBA{240, 100, 100, 255}
		return nil, err
	}

	if result.IssueURL != "" {
		s.statusMsg = fmt.Sprintf("Report sent! Issue: %s", result.IssueURL)
		s.statusColor = color.RGBA{100, 240, 120, 255}
	} else if result.LocalFilePath != "" {
		s.statusMsg = fmt.Sprintf("Report saved locally: %s", result.LocalFilePath)
		s.statusColor = color.RGBA{120, 220, 240, 255}
	} else {
		s.statusMsg = result.Message
		s.statusColor = color.RGBA{100, 240, 120, 255}
	}

	return result, nil
}

func (s *BugReportScreen) triggerSubmit() {
	if s.submitting {
		return
	}
	go func() {
		_, _ = s.SubmitSync()
	}()
}

func (s *BugReportScreen) triggerCopyAndSave() {
	report := bugreport.CollectReport(
		s.level,
		screenui.ScreenNameToString(s.prevScreenName),
		s.prevScreen,
		s.textInput.Text(),
	)

	localSubmitter := bugreport.NewLocalFileSubmitter("", nil)
	result, err := localSubmitter.Submit(report)
	if err != nil {
		s.statusMsg = fmt.Sprintf("Failed to save report: %v", err)
		s.statusColor = color.RGBA{240, 100, 100, 255}
		return
	}

	s.statusMsg = fmt.Sprintf("Saved to %s (Copied to clipboard)", result.LocalFilePath)
	s.statusColor = color.RGBA{120, 240, 150, 255}
}

func (s *BugReportScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.PopScr, nil, nil
	}

	s.textInput.Update(scale)

	opts := &ebiten.DrawImageOptions{}
	for _, b := range s.buttons {
		b.Update(opts, scale, W, H)
		if !b.IsClicked() {
			continue
		}
		switch b.ID {
		case "submit":
			s.triggerSubmit()
		case "copy_save":
			s.triggerCopyAndSave()
		case "close":
			return screenui.PopScr, nil, nil
		}
	}

	return screenui.BugReportScr, nil, nil
}

func (s *BugReportScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(bugPanelX)*scale, float64(bugPanelY)*scale)
	screen.DrawImage(s.panelBg, opts)

	vector.StrokeRect(
		screen,
		float32(float64(bugPanelX)*scale),
		float32(float64(bugPanelY)*scale),
		float32(float64(bugPanelW)*scale),
		float32(float64(bugPanelH)*scale),
		2*float32(scale),
		color.RGBA{200, 170, 100, 255},
		false,
	)

	titleFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 22}
	titleOpts := &text.DrawOptions{}
	titleOpts.GeoM.Translate(float64(bugPanelX+30)*scale, float64(bugPanelY+25)*scale)
	titleOpts.ColorScale.ScaleWithColor(color.RGBA{250, 230, 140, 255})
	text.Draw(screen, "Report a Bug / Issue", titleFace, titleOpts)

	subtitleFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 14}
	subOpts := &text.DrawOptions{}
	subOpts.GeoM.Translate(float64(bugPanelX+30)*scale, float64(bugPanelY+58)*scale)
	subOpts.ColorScale.ScaleWithColor(color.RGBA{180, 170, 160, 255})
	subText := fmt.Sprintf("Screen: %s | Includes game state & duel history", screenui.ScreenNameToString(s.prevScreenName))
	text.Draw(screen, subText, subtitleFace, subOpts)

	s.textInput.Draw(screen, scale)

	if s.statusMsg != "" {
		statusFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 13}
		stOpts := &text.DrawOptions{}
		stOpts.GeoM.Translate(float64(bugPanelX+30)*scale, float64(bugPanelY+380)*scale)
		stOpts.ColorScale.ScaleWithColor(s.statusColor)
		text.Draw(screen, s.statusMsg, statusFace, stOpts)
	}

	for _, b := range s.buttons {
		b.Draw(screen, opts, scale)
	}
}
