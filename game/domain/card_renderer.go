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
	"sync"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	frameCache     sync.Map
	manaIconsCache []*ebiten.Image
	setIconsCache  []*ebiten.Image
	initIconsOnce  sync.Once
	resizedCache   sync.Map
)

type resizedCardKey struct {
	id    string
	width int
	view  CardView
}

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

// GetFrameFilename determines the vintage card frame filename for a card.
func GetFrameFilename(card *Card) string {
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

// loadFrameImage loads and caches a card frame from embedded assets.
func loadFrameImage(filename string) image.Image {
	if cached, ok := frameCache.Load(filename); ok {
		return cached.(image.Image)
	}

	data, err := assets.CardFramesFS.ReadFile("art/card/" + filename)
	if err != nil {
		fmt.Printf("WARN: Failed to read frame %s: %v\n", filename, err)
		return nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		fmt.Printf("WARN: Failed to decode frame %s: %v\n", filename, err)
		return nil
	}

	frameCache.Store(filename, img)
	return img
}

func initIcons() {
	initIconsOnce.Do(func() {
		// Load mana symbols
		manaData, err := assets.CardFramesFS.ReadFile("art/card/Manasymbols.pic.png")
		if err == nil {
			if sheet, _, decodeErr := image.Decode(bytes.NewReader(manaData)); decodeErr == nil {
				manaIconsCache = make([]*ebiten.Image, 19)
				for i := range 19 {
					iconRGBA := image.NewRGBA(image.Rect(0, 0, 18, 18))
					for y := range 18 {
						for x := range 18 {
							srcX := i*18 + x
							srcColor := color.RGBAModel.Convert(sheet.At(srcX, y)).(color.RGBA)
							dx := float64(x) - 8.5
							dy := float64(y) - 8.5
							dist := dx*dx + dy*dy
							if dist <= 64 && (srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0) {
								srcColor.A = 255
								iconRGBA.Set(x, y, srcColor)
							} else if dist <= 72 && (srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0) {
								alpha := uint8(255 * (1.0 - (dist-64.0)/8.0))
								srcColor.A = alpha
								iconRGBA.Set(x, y, srcColor)
							}
						}
					}
					manaIconsCache[i] = ebiten.NewImageFromImage(iconRGBA)
				}
			}
		}

		// Load set symbols
		setData, err := assets.CardFramesFS.ReadFile("art/card/Cardsets.pic.png")
		if err == nil {
			if sheet, _, decodeErr := image.Decode(bytes.NewReader(setData)); decodeErr == nil {
				setIconsCache = make([]*ebiten.Image, 22)
				for i := range 22 {
					iconRGBA := image.NewRGBA(image.Rect(0, 0, 15, 15))
					for y := range 15 {
						for x := range 15 {
							srcX := i*15 + x
							srcColor := color.RGBAModel.Convert(sheet.At(srcX, y)).(color.RGBA)
							if srcColor.R != 0 || srcColor.G != 0 || srcColor.B != 0 {
								srcColor.A = 255
								iconRGBA.Set(x, y, srcColor)
							}
						}
					}
					setIconsCache[i] = ebiten.NewImageFromImage(iconRGBA)
				}
			}
		}
	})
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

	// Native art window on 228x325 frame: (21, 38, 207, 156) -> width=186, height=118
	scaleX := float64(w) / 228.0
	scaleY := float64(h) / 325.0

	artX1 := int(math.Round(21.0 * scaleX))
	artY1 := int(math.Round(38.0 * scaleY))
	artX2 := int(math.Round(207.0 * scaleX))
	artY2 := int(math.Round(156.0 * scaleY))

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

// RenderResizedCard builds a sharp, scaled card image at targetW with native text rendering.
func RenderResizedCard(card *Card, targetW int, view CardView) *ebiten.Image {
	if card == nil || targetW <= 0 {
		return nil
	}

	key := resizedCardKey{id: card.cardID, width: targetW, view: view}
	if cached, ok := resizedCache.Load(key); ok {
		return cached.(*ebiten.Image)
	}

	initIcons()

	frameFn := GetFrameFilename(card)
	frameImg := loadFrameImage(frameFn)
	if frameImg == nil {
		return nil
	}

	scale := float64(targetW) / 228.0
	var nativeCropH int
	if view == CardViewArtOnly {
		nativeCropH = 176
	} else {
		nativeCropH = 325
	}

	targetH := int(math.Round(float64(nativeCropH) * scale))
	if targetH <= 0 {
		targetH = 1
	}

	// 1. Crop frame to requested view
	var frameCrop image.Image
	if sub, ok := frameImg.(interface {
		SubImage(r image.Rectangle) image.Image
	}); ok {
		frameCrop = sub.SubImage(image.Rect(0, 0, 228, nativeCropH))
	} else {
		cropRGBA := image.NewRGBA(image.Rect(0, 0, 228, nativeCropH))
		draw.Draw(cropRGBA, cropRGBA.Bounds(), frameImg, image.Point{}, draw.Src)
		frameCrop = cropRGBA
	}

	// Scale frame
	scaledFrame := imageutil.ScaleImage(ebiten.NewImageFromImage(frameCrop), scale)
	resImg := ebiten.NewImage(targetW, targetH)
	resImg.DrawImage(scaledFrame, &ebiten.DrawImageOptions{})

	// 2. Extract and scale Art
	fullCardImg, _ := card.CardImage(CardViewFull)
	if fullCardImg != nil {
		rawArt := ExtractArtSubImage(fullCardImg)
		if rawArt != nil {
			artTargetW := int(math.Round(186.0 * scale))
			artTargetH := int(math.Round(118.0 * scale))
			artTargetX := int(math.Round(21.0 * scale))
			artTargetY := int(math.Round(38.0 * scale))

			if artTargetW > 0 && artTargetH > 0 {
				artEbiten := ebiten.NewImageFromImage(rawArt)
				artScaleX := float64(artTargetW) / float64(artEbiten.Bounds().Dx())
				artScaleY := float64(artTargetH) / float64(artEbiten.Bounds().Dy())

				artOpts := &ebiten.DrawImageOptions{}
				artOpts.GeoM.Scale(artScaleX, artScaleY)
				artOpts.GeoM.Translate(float64(artTargetX), float64(artTargetY))
				resImg.DrawImage(artEbiten, artOpts)
			}
		}
	}

	isBlackFrame := strings.Contains(frameFn, "Cardbk_Black.pic.png")
	var headerTextColor color.Color = color.Black
	if isBlackFrame {
		headerTextColor = color.White
	}

	// 3. Mana Cost (drawn right-to-left in title bar)
	currX := float64(int(math.Round(204.0 * scale)))
	manaY := float64(int(math.Round(18.0 * scale)))
	pipSize := math.Max(8.0, math.Round(14.0*scale))

	tokens := manaTokenRegex.FindAllStringSubmatch(card.ManaCost, -1)
	if len(manaIconsCache) > 0 {
		for _, token := range slices.Backward(tokens) {
			tok := strings.ToUpper(token[1])
			if iconIdx, ok := tokenToManaIcon[tok]; ok && iconIdx < len(manaIconsCache) {
				icon := manaIconsCache[iconIdx]
				if icon != nil {
					currX -= pipSize
					iconScale := pipSize / float64(icon.Bounds().Dx())
					opts := &ebiten.DrawImageOptions{}
					opts.GeoM.Scale(iconScale, iconScale)
					opts.GeoM.Translate(currX, manaY)
					resImg.DrawImage(icon, opts)
					currX -= 1.0
				}
			}
		}
	}

	// 4. Card Title
	titleX := float64(int(math.Round(24.0 * scale)))
	titleY := float64(int(math.Round(18.0 * scale)))
	maxTitleW := currX - titleX - 2.0

	titleFontSize := math.Max(7.0, math.Round(13.0*scale))
	titleFace := &text.GoTextFace{
		Source: fonts.MtgFont,
		Size:   titleFontSize,
	}

	for titleFontSize > 6.0 {
		w, _ := text.Measure(card.CardName, titleFace, 0)
		if w <= maxTitleW {
			break
		}
		titleFontSize -= 0.5
		titleFace.Size = titleFontSize
	}

	drawTextOnImage(resImg, card.CardName, titleFace, titleX, titleY, headerTextColor)

	// 5. Type Line & Set Icon (if included in bounds)
	if targetH >= int(math.Round(170.0*scale)) {
		typeX := float64(int(math.Round(24.0 * scale)))
		typeY := float64(int(math.Round(162.0 * scale)))
		typeLine := strings.ReplaceAll(strings.ReplaceAll(card.TypeLine, "\u2014", "-"), "—", "-")

		// Set icon
		setID := strings.ToLower(card.SetID)
		if setIdx, ok := setToSetIcon[setID]; ok && setIdx < len(setIconsCache) {
			sIcon := setIconsCache[setIdx]
			if sIcon != nil {
				sIconSize := math.Max(8.0, math.Round(13.0*scale))
				sIconScale := sIconSize / float64(sIcon.Bounds().Dx())
				sX := float64(int(math.Round(192.0 * scale)))
				sY := float64(int(math.Round(161.0 * scale)))
				sOpts := &ebiten.DrawImageOptions{}
				sOpts.GeoM.Scale(sIconScale, sIconScale)
				sOpts.GeoM.Translate(sX, sY)
				resImg.DrawImage(sIcon, sOpts)
			}
		}

		typeFontSize := math.Max(6.5, math.Round(10.5*scale))
		typeFace := &text.GoTextFace{
			Source: fonts.MtgFont,
			Size:   typeFontSize,
		}
		drawTextOnImage(resImg, typeLine, typeFace, typeX, typeY, headerTextColor)
	}

	resizedCache.Store(key, resImg)
	return resImg
}

func drawTextOnImage(dst *ebiten.Image, txt string, fontFace *text.GoTextFace, x, y float64, clr color.Color) {
	opts := text.DrawOptions{}
	opts.GeoM.Translate(x, y)
	r, g, b, a := clr.RGBA()
	opts.ColorScale.Scale(float32(r)/65535, float32(g)/65535, float32(b)/65535, float32(a)/65535)
	text.Draw(dst, txt, fontFace, &opts)
}

// ClearResizedCardCache clears the resized card cache.
func ClearResizedCardCache() {
	resizedCache.Clear()
}

// InvalidateResizedCardCache clears cached resized versions for a given card ID.
func InvalidateResizedCardCache(id string) {
	resizedCache.Range(func(k, v any) bool {
		if key, ok := k.(resizedCardKey); ok && key.id == id {
			resizedCache.Delete(key)
		}
		return true
	})
}
