package domain

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Token to mana symbol index in Manasymbols.pic.png (19 icons of 18x18)
var tokenToManaIcon = map[string]int{
	"X": 0, "0": 1, "1": 2, "2": 3, "3": 4, "4": 5, "5": 6, "6": 7, "7": 8, "8": 9, "9": 10,
	"10": 11, "W": 12, "R": 13, "U": 14, "B": 15, "G": 16, "T": 17, "C": 1,
}

// Set ID to set expansion icon index in Cardsets.pic.png (22 icons of 15x15)
var setToSetIcon = map[string]int{
	"arn": 1, "atq": 10, "leg": 9, "drk": 5, "fem": 6, "ice": 18,
}

var manaTokenRegex = regexp.MustCompile(`\{([^}]+)\}`)

var cardRulesFont = newCardFont(assets.CardRegularFont)
var cardLabelFont = newCardFont(assets.CardLabelFont)
var cardFlavorFont = newCardFont(assets.CardItalicFont)

func newCardFont(data []byte) *text.GoTextFaceSource {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	return source
}

func cardContrastOverlay(view CardView, sx, sy float64) *image.RGBA {
	overlay := image.NewRGBA(image.Rect(0, 0, view.X, view.Y))
	border := max(2, int(math.Round(2*math.Min(sx, sy))))
	black := &image.Uniform{C: color.Black}
	for _, rect := range []image.Rectangle{
		image.Rect(0, 0, view.X, border), image.Rect(0, view.Y-border, view.X, view.Y),
		image.Rect(0, 0, border, view.Y), image.Rect(view.X-border, 0, view.X, view.Y),
	} {
		draw.Draw(overlay, rect, black, image.Point{}, draw.Src)
	}
	return overlay
}

// renderCardImage draws text at the requested image size.
// The original artwork files are **mostly 288 × 232 pixels**, rather than 189 × 147:
//
// - `MEDART.CAT`: 404 of 430 images are **288 × 232**; the others have heights of 224, 230, or 236.
// - `SMALLART.CAT`: 427 of 429 images are **144 × 116**; two have heights of 112 and 118.
//
// The renderer scales artwork and text using a **200 × 300 card coordinate system**. These offsets are measured from the card’s upper-left corner:
//
// | Element | Offset `(x, y)` | Rectangle size | Font / height |
// |---|---:|---:|---|
// | Artwork | `(21, 25)` | **160 × 139** | — |
// | Card name | `(12, 9)` | 176 × 13 | MagicMedieval, **16** |
// | Card type | `(12, 169)` | 174 × 11 | MagicMedieval, **14** |
// | Rules text | `(28, 185)` | **146 × 83** | MPZurich Cn BT, **14** |
// | Text panel background | `(20, 180)` | 162 × 91 | — |
//
// The rendering rectangles (src/magic/NedCard/Palette.c:3229) and original font settings (program/DuelArt/DUEL.DAT:1) specify these values. Font heights are GDI logical units, **not point sizes**. Name and type text are vertically centered within their rectangles and have a shadow offset of `(1, 1)`.
//
// For a displayed card of size `W × H`, scale horizontal values by `W/200` and vertical values by `H/300`. Thus, **189 × 147 could be a chosen display size**, but it is neither the stored artwork size nor the renderer’s base artwork rectangle.
//
// The rules-text offset can move upward when the “expand text box” option is enabled and the text overflows. Flavor text uses the same font in italics and follows the rules text.

func renderCardImage(card *Card, art image.Image, view CardView) *ebiten.Image {
	initIcons()
	frameImg := loadFrameImage(getFrameFilename(card))
	if frameImg == nil {
		return nil
	}
	artOnly := view == CardViewArtOnly || view == CardViewArtMini
	baseHeight := 300.0
	if artOnly {
		baseHeight = float64(cardSourceArtHeight)
	}
	sx, sy := float64(view.X)/200, float64(view.Y)/baseHeight
	result := ebiten.NewImage(view.X, view.Y)
	frameOpts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	frameOpts.GeoM.Scale(float64(view.X)/float64(frameImg.Bounds().Dx()), sy*300/float64(frameImg.Bounds().Dy()))
	result.DrawImage(ebiten.NewImageFromImage(frameImg), frameOpts)
	if art != nil && !art.Bounds().Empty() {
		opts := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		opts.GeoM.Scale(160*sx/float64(art.Bounds().Dx()), 139*sy/float64(art.Bounds().Dy()))
		opts.GeoM.Translate(21*sx, 25*sy)
		result.DrawImage(ebiten.NewImageFromImage(art), opts)
	}
	result.DrawImage(ebiten.NewImageFromImage(cardContrastOverlay(view, sx, sy)), nil)
	currX := 188.0
	tokens := manaTokenRegex.FindAllStringSubmatch(card.ManaCost, -1)
	for _, token := range slices.Backward(tokens) {
		idx, ok := tokenToManaIcon[strings.ToUpper(token[1])]
		if !ok || idx >= len(manaIconsCache) || manaIconsCache[idx] == nil {
			continue
		}
		icon := manaIconsCache[idx]
		currX -= 13
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(13*sx/float64(icon.Bounds().Dx()), 13*sy/float64(icon.Bounds().Dy()))
		opts.GeoM.Translate(currX*sx, 9*sy)
		result.DrawImage(icon, opts)
		currX--
	}
	drawCardLabel(result, card.CardName, 16, 12, 5, math.Max(0, currX-14), 17, sx, sy)
	if artOnly {
		return result
	}
	typeWidth := 174.0
	if idx, ok := setToSetIcon[strings.ToLower(card.SetID)]; ok && idx < len(setIconsCache) && setIconsCache[idx] != nil {
		icon := setIconsCache[idx]
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Scale(11*sx/float64(icon.Bounds().Dx()), 11*sy/float64(icon.Bounds().Dy()))
		opts.GeoM.Translate(175*sx, 169*sy)
		result.DrawImage(icon, opts)
		typeWidth -= 13
	}
	drawCardLabel(result, strings.ReplaceAll(card.TypeLine, "—", "-"), 12, 12, 166, typeWidth, 13, sx, sy)
	rules := card.Text
	if card.FlavorText != "" {
		if rules != "" {
			rules += "\n\n"
		}
		rules += card.FlavorText
	}
	face := &text.GoTextFace{Source: cardRulesFont, Size: 14}
	lines := wrapCardText(rules, face, 146)
	for face.Size > 5 && float64(len(lines))*face.Size*1.2 > 83 {
		face.Size -= 0.5
		lines = wrapCardText(rules, face, 146)
	}
	flavorStart := len(lines)
	if card.FlavorText != "" {
		flavorStart = 0
		if card.Text != "" {
			flavorStart = len(wrapCardText(card.Text, face, 146)) + 1
		}
	}
	for i, line := range lines {
		drawCardText(result, line, face, 28, 185+float64(i)*face.Size*1.2, sx, sy, i >= flavorStart)
	}
	if card.CardType == CardTypeCreature || strings.Contains(card.TypeLine, "Creature") {
		stat := func(value int) string {
			if value < 0 {
				return "*"
			}
			return fmt.Sprint(value)
		}
		face.Size = 13
		stats := stat(card.Power) + "/" + stat(card.Toughness)
		width, _ := text.Measure(stats, face, 0)
		drawTextOnImage(result, stats, face, 186-width, 278, sx, sy)
	}
	return result
}

func drawCardLabel(dst *ebiten.Image, value string, size, x, y, width, height, sx, sy float64) {
	face := &text.GoTextFace{Source: cardLabelFont, Size: size}
	for face.Size > 5 {
		w, _ := text.Measure(value, face, 0)
		if w <= width {
			break
		}
		face.Size -= 0.5
	}
	_, h := text.Measure(value, face, 0)
	drawTextOnImage(dst, value, face, x, y+(height-h)/2, sx, sy)
}

func wrapCardText(value string, face *text.GoTextFace, width float64) []string {
	var lines []string
	for _, paragraph := range strings.Split(value, "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			w, _ := text.Measure(candidate, face, 0)
			if w > width && line != "" {
				lines = append(lines, line)
				line = word
			} else {
				line = candidate
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// getFrameFilename determines the vintage card frame filename for a card.
func getFrameFilename(card *Card) string {
	typeLine := card.TypeLine
	colors := card.Colors
	setID := strings.ToLower(card.SetID)

	if strings.Contains(typeLine, "Land") {
		if strings.Contains(typeLine, "Plains") && !hasAny(typeLine, "Island", "Swamp", "Mountain", "Forest") {
			return "Cardbk_Whiteland.pic.png"
		}
		if strings.Contains(typeLine, "Island") && !hasAny(typeLine, "Plains", "Swamp", "Mountain", "Forest") {
			return "Cardbk_Blueland.pic.png"
		}
		if strings.Contains(typeLine, "Swamp") && !hasAny(typeLine, "Plains", "Island", "Mountain", "Forest") {
			return "Cardbk_Blackland.pic.png"
		}
		if strings.Contains(typeLine, "Mountain") && !hasAny(typeLine, "Plains", "Island", "Swamp", "Forest") {
			return "Cardbk_Redland.pic.png"
		}
		if strings.Contains(typeLine, "Forest") && !hasAny(typeLine, "Plains", "Island", "Swamp", "Mountain") {
			return "Cardbk_Greenland.pic.png"
		}

		switch setID {
		case "atq":
			return "Cardbk_Antiquitiesland.pic.png"
		case "arn":
			return "Cardbk_Arabiannightsland.pic.png"
		case "drk":
			return "Cardbk_Darklandsland.pic.png"
		case "fem":
			return "Cardbk_Fallenempiresland.pic.png"
		case "leg":
			return "Cardbk_Legendsland.pic.png"
		case "ice":
			return "Cardbk_Iceageland.pic.png"
		default:
			return "Cardbk_Antiquitiesland.pic.png"
		}
	}

	if strings.Contains(typeLine, "Artifact") {
		return "Cardbk_Artifact.pic.png"
	}

	if len(colors) > 1 {
		return "Cardbk_Gold.pic.png"
	}

	if len(colors) == 1 {
		switch colors[0] {
		case "W":
			return "Cardbk_White.pic.png"
		case "U":
			return "Cardbk_Blue.pic.png"
		case "B":
			return "Cardbk_Black.pic.png"
		case "R":
			return "Cardbk_Red.pic.png"
		case "G":
			return "Cardbk_Green.pic.png"
		}
	}

	return "Cardbk_Special.pic.png"
}

func hasAny(str string, substrs ...string) bool {
	for _, s := range substrs {
		if strings.Contains(str, s) {
			return true
		}
	}
	return false
}

// ExtractArtSubImage crops the pure artwork from a full card image.
func ExtractArtSubImage(fullImg image.Image) image.Image {
	if fullImg == nil {
		return nil
	}
	bounds := fullImg.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w == 0 || h == 0 {
		return nil
	}

	// Use the same art rectangle as the card renderer.
	scaleX := float64(w) / 200.0
	scaleY := float64(h) / 300.0

	artX1 := int(math.Round(21.0 * scaleX))
	artY1 := int(math.Round(25.0 * scaleY))
	artX2 := int(math.Round(181.0 * scaleX))
	artY2 := int(math.Round(164.0 * scaleY))

	if artX2 > w {
		artX2 = w
	}
	if artY2 > h {
		artY2 = h
	}

	subRect := image.Rect(artX1, artY1, artX2, artY2)
	if sub, ok := fullImg.(interface {
		SubImage(r image.Rectangle) image.Image
	}); ok {
		return sub.SubImage(subRect)
	}

	// Fallback copy
	artRGBA := image.NewRGBA(image.Rect(0, 0, subRect.Dx(), subRect.Dy()))
	draw.Draw(artRGBA, artRGBA.Bounds(), fullImg, subRect.Min, draw.Src)
	return artRGBA
}

func drawTextOnImage(dst *ebiten.Image, txt string, fontFace *text.GoTextFace, x, y, sx, sy float64) {
	drawCardText(dst, txt, fontFace, x, y, sx, sy, false)
}

func drawCardText(dst *ebiten.Image, txt string, fontFace *text.GoTextFace, x, y, sx, sy float64, italic bool) {
	var transform ebiten.GeoM
	if italic {
		italicFace := *fontFace
		italicFace.Source = cardFlavorFont
		fontFace = &italicFace
	}
	transform.Scale(sx, sy)
	opts := text.DrawOptions{}
	opts.GeoM = transform
	opts.GeoM.Translate(x*sx, y*sy)
	opts.ColorScale.Scale(0, 0, 0, 1)
	text.Draw(dst, txt, fontFace, &opts)
}
