package screens

import (
	"fmt"
	"image/color"
	"math/rand"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

type WisemanScreen struct {
	BgImage      *ebiten.Image
	CurrentStory []string // Paginated story
	Page         int
}

func (s *WisemanScreen) IsFramed() bool {
	return false
}

func NewWisemanScreen() *WisemanScreen {
	bgImg, err := imageutil.LoadImage(assets.Wiseman_png)
	if err != nil {
		panic(fmt.Sprintf("Unable to load Wiseman.png: %s", err))
	}

	stories := loadStories()
	story := "No stories found."
	if len(stories) > 0 {
		story = stories[rand.Intn(len(stories))]
	}

	// Pagination
	// Area is {0,0} to {290, 768}.
	// We need to wrap text and then split into pages if it exceeds height.
	fontFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   24,
	}
	paginatedStory := paginateText(story, fontFace, 290, 768)

	return &WisemanScreen{
		BgImage:      bgImg,
		CurrentStory: paginatedStory,
		Page:         0,
	}
}

func (s *WisemanScreen) Update(W, H int, scale float64) (screenui.ScreenName, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {

		s.Page++
		if s.Page >= len(s.CurrentStory) {
			return screenui.CityScr, nil
		}
	}
	return screenui.WisemanScr, nil
}

func (s *WisemanScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	// Draw background scaled to screen size (1024x768)
	bgOpts := &ebiten.DrawImageOptions{}
	bgW, bgH := s.BgImage.Bounds().Dx(), s.BgImage.Bounds().Dy()
	scaleX := float64(1024) / float64(bgW)
	scaleY := float64(768) / float64(bgH)

	// Apply global scale if needed, but the requirement says "scale wiseman image to screen size (1024x768)"
	// The Draw method receives W, H and scale. Usually W, H are the logical screen size (1024x768 * scale).
	// But let's stick to the logical 1024x768 and let the game loop handle the final scaling if it does.
	// Wait, looking at city.go:
	// cityOpts.GeoM.Scale(scale, scale)
	// cityOpts.GeoM.Scale(SCALE, SCALE)
	// It seems we should respect the passed `scale`.

	bgOpts.GeoM.Scale(scaleX, scaleY)
	bgOpts.GeoM.Scale(scale, scale)
	screen.DrawImage(s.BgImage, bgOpts)

	// Draw text
	if s.Page < len(s.CurrentStory) {
		textStr := s.CurrentStory[s.Page]

		// Text position: {0,0} to {290, 768}
		// Add padding
		const padding = 20

		txt := elements.NewText(24, textStr, int(padding), int(padding))
		txt.LineSpacing = 30.0
		txt.Color = color.White
		txt.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}
}

func loadStories() []string {
	content := string(assets.Advblocks_txt)
	var stories []string

	// Split by STARTBLOCK and ENDBLOCK
	// The file format is:
	// ... header ...
	// STARTBLOCK
	// ... story ...
	// ENDBLOCK

	parts := strings.Split(content, "STARTBLOCK")
	for _, part := range parts {
		end := strings.Index(part, "ENDBLOCK")
		if end != -1 {
			story := strings.TrimSpace(part[:end])
			if story != "" {
				stories = append(stories, story)
			}
		}
	}
	return stories
}

func paginateText(textStr string, face *text.GoTextFace, maxWidth, maxHeight float64) []string {
	var pages []string

	// First, wrap text to fit width
	words := strings.Fields(textStr)
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	currentLine := words[0]

	// Estimate width. text.Measure is not available in v2 directly on face?
	// text.Measure(text, face, spacing)
	// Actually `text.Measure` exists.

	for _, word := range words[1:] {
		newLine := currentLine + " " + word
		w, _ := text.Measure(newLine, face, 30.0)
		if w > maxWidth-40 { // 40 for padding
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = newLine
		}
	}
	lines = append(lines, currentLine)

	// Now split into pages based on height
	var currentPageLines []string
	currentHeight := 0.0
	lineHeight := 30.0 // Approximation

	for _, line := range lines {
		if currentHeight+lineHeight > maxHeight-40 {
			// Page full
			// Add ellipses to the last line of the previous page if it's not the end?
			// Requirement: "If the text is too large to fit in that rectangle add ellipses and use click/touch/spacebar to advance"
			// This implies we should just split it.

			// If we are splitting mid-sentence, maybe add "..."
			if len(currentPageLines) > 0 {
				currentPageLines[len(currentPageLines)-1] += "..."
			}

			pages = append(pages, strings.Join(currentPageLines, "\n"))
			currentPageLines = []string{line}
			currentHeight = lineHeight
		} else {
			currentPageLines = append(currentPageLines, line)
			currentHeight += lineHeight
		}
	}
	if len(currentPageLines) > 0 {
		pages = append(pages, strings.Join(currentPageLines, "\n"))
	}

	return pages
}
