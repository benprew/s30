package screens

import (
	"fmt"
	"image"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type startMode int

const (
	startModeMenu startMode = iota
	startModeLoad
	startModeDifficulty
	startModeColor
)

type StartScreen struct {
	mode               startMode
	background         *ebiten.Image
	menu2Bg            *ebiten.Image
	menu3Bg            *ebiten.Image
	newGameBtn         *elements.Button
	loadGameBtn        *elements.Button
	backBtn            *elements.Button
	saveButtons        []*elements.Button
	difficultyButtons  []*elements.Button
	colorButtons       []*elements.Button
	difficultyLabels   []string
	saves              []save.SaveInfo
	hasSaves           bool
	SelectedSave       string
	SelectedDifficulty domain.Difficulty
	SelectedColor      domain.ColorMask
	NewGame            bool
}

func (s *StartScreen) IsFramed() bool { return false }

func (s *StartScreen) IsOverlay() bool { return false }

// virtScreenW/H mirror Game.Layout — the logical canvas size all screens use.
const virtScreenW = 1024
const virtScreenH = 768

var difficultyOrder = []domain.Difficulty{
	domain.DifficultyEasy,   // Apprentice
	domain.DifficultyMedium, // Magician
	domain.DifficultyHard,   // Sorcerer
	domain.DifficultyExpert, // Wizard
}

// difficultyLabelText is parallel to difficultyOrder, sharing the same names
// used in save files via domain.DifficultyToString.
var difficultyLabelText = func() []string {
	labels := make([]string, len(difficultyOrder))
	for i, d := range difficultyOrder {
		labels[i] = domain.DifficultyToString(d)
	}
	return labels
}()

type colorBlurb struct {
	Name string
	Desc string
}

// colorBlurbs is parallel to colorOrder.
var colorBlurbs = []colorBlurb{
	{"Red", "Red mages channel the fury of chaos - fire, storm, and the wild heart of battle."},
	{"White", "White magic walks the path of light - healing, protection, and the honored arts of war."},
	{"Black", "Black magic whispers from the grave, drawing power from death, decay, and forbidden pacts."},
	{"Green", "Green magic speaks with nature's voice, soothing as still water, savage as the storm."},
	{"Blue", "Blue magic weaves thought into substance - the realm of intellect, artifice, and illusion."},
}

// colorOrder matches the visual order of the Menu3 button sprite sheet.
var colorOrder = []domain.ColorMask{
	domain.ColorRed,
	domain.ColorWhite,
	domain.ColorBlack,
	domain.ColorGreen,
	domain.ColorBlue,
}

func NewStartScreen() *StartScreen {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(fmt.Sprintf("Error loading button sprites: %v", err))
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 24}

	newGameW, newGameH := elements.TextButtonSize("New Game", fontFace)
	loadGameW, _ := elements.TextButtonSize("Load Game", fontFace)

	centerX := 512
	btnY := 400

	s := &StartScreen{
		SelectedDifficulty: domain.DifficultyEasy,
		SelectedColor:      domain.ColorColorless,
	}

	s.background = scaledFullScreen(assets.StartTitle_png)
	s.menu2Bg = scaledFullScreen(assets.StartMenu2_png)
	s.menu3Bg = scaledFullScreen(assets.StartMenu3_png)

	hasSaves := false
	if saveDir, err := save.SaveDir(); err == nil {
		if saves, err := save.ListSaves(saveDir); err == nil && len(saves) > 0 {
			hasSaves = true
		}
	}
	s.hasSaves = hasSaves

	s.newGameBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "New Game",
		Font:    fontFace,
		ID:      "new_game",
		X:       centerX - newGameW/2,
		Y:       btnY,
	})

	s.loadGameBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Load Game",
		Font:    fontFace,
		ID:      "load_game",
		X:       centerX - loadGameW/2,
		Y:       btnY + newGameH + 20,
	})

	backW, _ := elements.TextButtonSize("Back", fontFace)
	s.backBtn = elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal:  btnSprites[0][0],
		Hover:   btnSprites[0][1],
		Pressed: btnSprites[0][2],
		Text:    "Back",
		Font:    fontFace,
		ID:      "back",
		X:       centerX - backW/2,
		Y:       650,
	})

	// Menu2 background has warrior portrait on the left, so difficulty buttons
	// go on the right; Menu3 has the dragon on the right, buttons on the left.
	// The Menu2 background paints four drop-shadow slots at these centers
	// (1024x768 virtual coords); buttons are scaled and placed to fill them.
	s.difficultyButtons = buildIconButtons(iconButtonOpts{
		normalAsset: assets.StartMenu2Norm_png,
		hoverAsset:  assets.StartMenu2Hi_png,
		n:           len(difficultyOrder),
		idPrefix:    "difficulty",
		centerX:     826,
		scale:       1.5,
		slotCenters: []int{188, 349, 507, 667},
	})
	s.colorButtons = buildIconButtons(iconButtonOpts{
		normalAsset: assets.StartMenu3But1_png,
		hoverAsset:  assets.StartMenu3but_png,
		n:           len(colorOrder),
		idPrefix:    "color",
		centerX:     110,
		scale:       1.8,
	})
	s.difficultyLabels = difficultyLabelText

	return s
}

// scaledFullScreen loads a 640x480 .pic.png background and scales it to fill
// the 1024x768 virtual canvas.
func scaledFullScreen(asset []byte) *ebiten.Image {
	img, err := imageutil.LoadImage(asset)
	if err != nil {
		panic(fmt.Sprintf("Error loading background: %v", err))
	}
	b := img.Bounds()
	scaleX := float64(virtScreenW) / float64(b.Dx())
	scaleY := float64(virtScreenH) / float64(b.Dy())
	return imageutil.ScaleImageInd(img, scaleX, scaleY)
}

type iconButtonOpts struct {
	normalAsset, hoverAsset []byte
	n                       int
	idPrefix                string
	centerX                 int
	scale                   float64 // 1.0 = native sheet size
	// slotCenters, if non-nil, places each button's vertical center at the
	// corresponding Y. Length must equal n. When nil, buttons are stacked
	// contiguously and vertically centered on the virtual canvas.
	slotCenters []int
}

// buildIconButtons slices a vertically-stacked sprite sheet into n button
// images. If opts.scale != 1.0 the sheet is rescaled before slicing.
func buildIconButtons(opts iconButtonOpts) []*elements.Button {
	normSheet, err := imageutil.LoadImage(opts.normalAsset)
	if err != nil {
		panic(fmt.Sprintf("Error loading %s normal sprites: %v", opts.idPrefix, err))
	}
	hiSheet, err := imageutil.LoadImage(opts.hoverAsset)
	if err != nil {
		panic(fmt.Sprintf("Error loading %s hover sprites: %v", opts.idPrefix, err))
	}

	if opts.scale != 0 && opts.scale != 1.0 {
		normSheet = imageutil.ScaleImage(normSheet, opts.scale)
		hiSheet = imageutil.ScaleImage(hiSheet, opts.scale)
	}

	sheetB := normSheet.Bounds()
	btnW := sheetB.Dx()
	btnH := sheetB.Dy() / opts.n
	x := opts.centerX - btnW/2

	startY := (virtScreenH - btnH*opts.n) / 2

	btns := make([]*elements.Button, opts.n)
	for i := range opts.n {
		rect := image.Rect(0, i*btnH, btnW, (i+1)*btnH)
		normSub := normSheet.SubImage(rect).(*ebiten.Image)
		hiSub := hiSheet.SubImage(rect).(*ebiten.Image)

		y := startY + i*btnH
		if i < len(opts.slotCenters) {
			y = opts.slotCenters[i] - btnH/2
		}

		btns[i] = elements.NewButtonFromConfig(elements.ButtonConfig{
			Normal:  normSub,
			Hover:   hiSub,
			Pressed: hiSub,
			ID:      fmt.Sprintf("%s_%d", opts.idPrefix, i),
			X:       x,
			Y:       y,
		})
	}
	return btns
}

func (s *StartScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	opts := &ebiten.DrawImageOptions{}

	switch s.mode {
	case startModeMenu:
		s.newGameBtn.Update(opts, scale, W, H)
		if s.newGameBtn.IsClicked() {
			s.NewGame = true
			s.mode = startModeDifficulty
			s.newGameBtn.State = elements.StateNormal
			return screenui.StartScr, nil, nil
		}

		if s.hasSaves {
			s.loadGameBtn.Update(opts, scale, W, H)
			if s.loadGameBtn.IsClicked() {
				s.loadSaveList()
			}
		}

	case startModeLoad:
		for _, btn := range s.saveButtons {
			btn.Update(opts, scale, W, H)
		}
		s.backBtn.Update(opts, scale, W, H)

		for i, btn := range s.saveButtons {
			if btn.IsClicked() {
				s.SelectedSave = s.saves[i].Path
				return screenui.WorldScr, nil, nil
			}
		}

		if s.backBtn.IsClicked() {
			s.mode = startModeMenu
			s.backBtn.State = elements.StateNormal
		}

	case startModeDifficulty:
		for i, btn := range s.difficultyButtons {
			btn.Update(opts, scale, W, H)
			if btn.IsClicked() {
				s.SelectedDifficulty = difficultyOrder[i]
				s.mode = startModeColor
				btn.State = elements.StateNormal
				return screenui.StartScr, nil, nil
			}
		}

	case startModeColor:
		for i, btn := range s.colorButtons {
			btn.Update(opts, scale, W, H)
			if btn.IsClicked() {
				s.SelectedColor = colorOrder[i]
				btn.State = elements.StateNormal
				return screenui.WorldScr, nil, nil
			}
		}
	}

	return screenui.StartScr, nil, nil
}

func (s *StartScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	opts := &ebiten.DrawImageOptions{}

	switch s.mode {
	case startModeMenu:
		screen.DrawImage(s.background, &ebiten.DrawImageOptions{})
		s.newGameBtn.Draw(screen, opts, scale)
		if s.hasSaves {
			s.loadGameBtn.Draw(screen, opts, scale)
		}

	case startModeLoad:
		screen.DrawImage(s.background, &ebiten.DrawImageOptions{})
		headerFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 30}
		headerText := "Load Game"
		headerW, _ := text.Measure(headerText, headerFont, 0)
		headerOpts := &text.DrawOptions{}
		headerOpts.GeoM.Translate(float64(W)/2-headerW/2, 280)
		headerOpts.ColorScale.Scale(1, 1, 1, 1)
		text.Draw(screen, headerText, headerFont, headerOpts)

		if len(s.saveButtons) == 0 {
			noSavesFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 20}
			noSavesText := "No saved games found"
			noSavesW, _ := text.Measure(noSavesText, noSavesFont, 0)
			noSavesOpts := &text.DrawOptions{}
			noSavesOpts.GeoM.Translate(float64(W)/2-noSavesW/2, 380)
			noSavesOpts.ColorScale.Scale(0.7, 0.7, 0.7, 1)
			text.Draw(screen, noSavesText, noSavesFont, noSavesOpts)
		}

		for _, btn := range s.saveButtons {
			btn.Draw(screen, opts, scale)
		}

		s.backBtn.Draw(screen, opts, scale)

	case startModeDifficulty:
		screen.DrawImage(s.menu2Bg, &ebiten.DrawImageOptions{})

		titleFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 56}
		titleText := "Select Difficulty Level"
		titleW, _ := text.Measure(titleText, titleFont, 0)
		titleOpts := &text.DrawOptions{}
		titleOpts.GeoM.Translate(float64(W)/2-titleW/2, 40)
		titleOpts.ColorScale.Scale(1, 1, 1, 1)
		text.Draw(screen, titleText, titleFont, titleOpts)

		labelFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 44}
		// Each button image has ~31px of transparent (formerly black) padding on
		// the left after the 1.5x scale; subtract that so labels sit close to the
		// visible button edge.
		const buttonLeftPad = 31
		for i, btn := range s.difficultyButtons {
			btn.Draw(screen, opts, scale)
			if i >= len(s.difficultyLabels) {
				continue
			}
			label := s.difficultyLabels[i]
			labelW, labelH := text.Measure(label, labelFont, 0)
			lx := float64(btn.Bounds.Min.X+buttonLeftPad) - labelW - 24
			ly := float64(btn.Bounds.Min.Y) + float64(btn.Bounds.Dy())/2 - labelH/2
			lblOpts := &text.DrawOptions{}
			lblOpts.GeoM.Translate(lx, ly)
			lblOpts.ColorScale.Scale(1, 1, 1, 1)
			text.Draw(screen, label, labelFont, lblOpts)
		}

	case startModeColor:
		screen.DrawImage(s.menu3Bg, &ebiten.DrawImageOptions{})

		nameFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 32}
		descFont := &text.GoTextFace{Source: fonts.MtgFont, Size: 22}
		// Wrap descriptions to fit between the icon column and the dragon panel.
		const textX = 200
		const textMaxW = 510 // dragon panel starts around x=723
		const lineSpacing = 6.0
		_, descLineH := text.Measure("Mg", descFont, 0)
		_, nameLineH := text.Measure("Mg", nameFont, 0)

		for i, btn := range s.colorButtons {
			btn.Draw(screen, opts, scale)
			if i >= len(colorBlurbs) {
				continue
			}
			b := colorBlurbs[i]
			lines := wrapText(b.Desc, descFont, textMaxW)
			blockH := nameLineH + (descLineH+lineSpacing)*float64(len(lines))
			btnCenterY := float64(btn.Bounds.Min.Y) + float64(btn.Bounds.Dy())/2
			startTextY := btnCenterY - blockH/2

			nameOpts := &text.DrawOptions{}
			nameOpts.GeoM.Translate(textX, startTextY)
			nameOpts.ColorScale.Scale(1, 1, 1, 1)
			text.Draw(screen, b.Name+":", nameFont, nameOpts)

			ly := startTextY + nameLineH
			for _, line := range lines {
				lineOpts := &text.DrawOptions{}
				lineOpts.GeoM.Translate(textX, ly)
				lineOpts.ColorScale.Scale(1, 1, 1, 1)
				text.Draw(screen, line, descFont, lineOpts)
				ly += descLineH + lineSpacing
			}
		}
	}
}

// wrapText breaks s into lines that each measure at most maxW with the given
// font, splitting on whitespace and never breaking words.
func wrapText(s string, face text.Face, maxW float64) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		candidate := cur + " " + w
		cw, _ := text.Measure(candidate, face, 0)
		if cw > maxW {
			lines = append(lines, cur)
			cur = w
		} else {
			cur = candidate
		}
	}
	lines = append(lines, cur)
	return lines
}

func (s *StartScreen) loadSaveList() {
	s.mode = startModeLoad

	saveDir, err := save.SaveDir()
	if err != nil {
		fmt.Printf("Error getting save directory: %v\n", err)
		return
	}

	saves, err := save.ListSaves(saveDir)
	if err != nil {
		fmt.Printf("Error listing saves: %v\n", err)
		return
	}

	s.saves = saves
	s.saveButtons = nil

	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		fmt.Printf("Error loading button sprites: %v\n", err)
		return
	}
	fontFace := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}

	maxVisible := min(len(saves), 8)

	centerX := 512
	startY := 330

	for i := range maxVisible {
		sv := saves[i]
		label := fmt.Sprintf("%s  -  %s", sv.Name, sv.SavedAt.Format("Jan 02 2006 15:04"))
		btnW, btnH := elements.TextButtonSize(label, fontFace)

		btn := elements.NewButtonFromConfig(elements.ButtonConfig{
			Normal:  btnSprites[0][0],
			Hover:   btnSprites[0][1],
			Pressed: btnSprites[0][2],
			Text:    label,
			Font:    fontFace,
			ID:      fmt.Sprintf("save_%d", i),
			X:       centerX - btnW/2,
			Y:       startY + i*(btnH+10),
		})
		s.saveButtons = append(s.saveButtons, btn)
	}
}
