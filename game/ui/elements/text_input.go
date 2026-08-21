package elements

import (
	"image"
	"image/color"
	"strings"

	"github.com/benprew/s30/game/ui/fonts"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// TextInput provides an interactive text area widget for typing notes.
type TextInput struct {
	X, Y, W, H    int
	runes         []rune
	cursorPos     int
	placeholder   string
	Focused       bool
	Multiline     bool
	cursorBlink   int
	backspaceHold int
	bgImage       *ebiten.Image
}

// NewTextInput creates a new TextInput widget.
func NewTextInput(x, y, w, h int, placeholder string) *TextInput {
	bg := ebiten.NewImage(w, h)
	bg.Fill(color.RGBA{15, 12, 10, 240})

	return &TextInput{
		X:           x,
		Y:           y,
		W:           w,
		H:           h,
		placeholder: placeholder,
		Focused:     true,
		Multiline:   true,
		bgImage:     bg,
	}
}

// Text returns the current contents of the input.
func (ti *TextInput) Text() string {
	return string(ti.runes)
}

// SetText replaces the contents and moves the cursor to the end.
func (ti *TextInput) SetText(s string) {
	ti.runes = []rune(s)
	ti.cursorPos = len(ti.runes)
}

// AppendText appends text to the end of the input.
func (ti *TextInput) AppendText(s string) {
	ti.runes = append(ti.runes, []rune(s)...)
	ti.cursorPos = len(ti.runes)
}

// Clear empties the text input.
func (ti *TextInput) Clear() {
	ti.runes = nil
	ti.cursorPos = 0
}

// InsertChar inserts a rune at the current cursor position.
func (ti *TextInput) InsertChar(r rune) {
	if ti.cursorPos < 0 {
		ti.cursorPos = 0
	}
	if ti.cursorPos > len(ti.runes) {
		ti.cursorPos = len(ti.runes)
	}

	ti.runes = append(ti.runes[:ti.cursorPos], append([]rune{r}, ti.runes[ti.cursorPos:]...)...)
	ti.cursorPos++
}

// HandleBackspace removes the rune immediately preceding the cursor.
func (ti *TextInput) HandleBackspace() {
	if ti.cursorPos <= 0 || len(ti.runes) == 0 {
		return
	}
	ti.runes = append(ti.runes[:ti.cursorPos-1], ti.runes[ti.cursorPos:]...)
	ti.cursorPos--
}

// Update handles mouse focus and keyboard input.
func (ti *TextInput) Update(scale float64) bool {
	rect := image.Rect(
		int(float64(ti.X)*scale),
		int(float64(ti.Y)*scale),
		int(float64(ti.X+ti.W)*scale),
		int(float64(ti.Y+ti.H)*scale),
	)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		ti.Focused = image.Pt(mx, my).In(rect)
	}

	if !ti.Focused {
		return false
	}

	ti.cursorBlink = (ti.cursorBlink + 1) % 60

	chars := ebiten.AppendInputChars(nil)
	for _, r := range chars {
		if r >= 32 || r == '\n' || r == '\t' {
			ti.InsertChar(r)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter) {
		if ti.Multiline {
			ti.InsertChar('\n')
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		ti.HandleBackspace()
		ti.backspaceHold = 0
	} else if ebiten.IsKeyPressed(ebiten.KeyBackspace) {
		ti.backspaceHold++
		if ti.backspaceHold > 25 && ti.backspaceHold%4 == 0 {
			ti.HandleBackspace()
		}
	} else {
		ti.backspaceHold = 0
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
		if ti.cursorPos > 0 {
			ti.cursorPos--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
		if ti.cursorPos < len(ti.runes) {
			ti.cursorPos++
		}
	}

	return true
}

// Draw renders the text input box, placeholder or text, and blinking cursor.
func (ti *TextInput) Draw(screen *ebiten.Image, scale float64) {
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(float64(ti.X)*scale, float64(ti.Y)*scale)
	screen.DrawImage(ti.bgImage, opts)

	borderColor := color.RGBA{120, 100, 70, 200}
	if ti.Focused {
		borderColor = color.RGBA{220, 180, 80, 255}
	}
	vector.StrokeRect(
		screen,
		float32(float64(ti.X)*scale),
		float32(float64(ti.Y)*scale),
		float32(float64(ti.W)*scale),
		float32(float64(ti.H)*scale),
		1.5*float32(scale),
		borderColor,
		false,
	)

	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   16,
	}

	padX := 12
	padY := 12
	startX := float64(ti.X+padX) * scale
	startY := float64(ti.Y+padY) * scale

	if len(ti.runes) == 0 && ti.placeholder != "" {
		pOpts := &text.DrawOptions{}
		pOpts.GeoM.Translate(startX, startY)
		pOpts.ColorScale.ScaleWithColor(color.RGBA{130, 120, 110, 180})
		text.Draw(screen, ti.placeholder, fontFace, pOpts)
		return
	}

	lines := strings.Split(string(ti.runes), "\n")
	lineHeight := 22.0 * scale
	curLine := 0
	curCol := 0
	charCount := 0

	for i, line := range lines {
		lineY := startY + float64(i)*lineHeight
		if lineY > float64(ti.Y+ti.H-padY)*scale {
			break
		}

		tOpts := &text.DrawOptions{}
		tOpts.GeoM.Translate(startX, lineY)
		tOpts.ColorScale.ScaleWithColor(color.RGBA{240, 235, 220, 255})
		text.Draw(screen, line, fontFace, tOpts)

		lineLen := len([]rune(line))
		if ti.cursorPos >= charCount && ti.cursorPos <= charCount+lineLen {
			curLine = i
			curCol = ti.cursorPos - charCount
		}
		charCount += lineLen + 1
	}

	if ti.Focused && ti.cursorBlink < 35 {
		prefix := ""
		if curLine < len(lines) {
			lineRunes := []rune(lines[curLine])
			if curCol <= len(lineRunes) {
				prefix = string(lineRunes[:curCol])
			}
		}
		prefixW, _ := text.Measure(prefix, fontFace, 0)
		cursorX := startX + prefixW
		cursorY := startY + float64(curLine)*lineHeight

		vector.StrokeLine(
			screen,
			float32(cursorX),
			float32(cursorY),
			float32(cursorX),
			float32(cursorY+18*scale),
			1.5*float32(scale),
			color.RGBA{255, 230, 120, 255},
			false,
		)
	}
}
