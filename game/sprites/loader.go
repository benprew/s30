package sprites

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/png" // Import for PNG decoding side effects

	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// SubimageInfo holds the position and dimensions of a found subimage.
type SubimageInfo struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// LoadSpriteSheet loads the embedded SpriteSheet.
// sprWidth is the number of images horizontally in the sheet
// sprHeight is the number of images vertically in the sheet
// pixel size of a single sprite iamge is deteremined by the image width divided by sprWidth
func LoadSpriteSheet(sprWidth, sprHeight int, file []byte) ([][]*ebiten.Image, error) {
	sheet, err := elements.LoadImage(file)
	if err != nil {
		return nil, err
	}

	bounds := sheet.Bounds()
	width := bounds.Max.X - bounds.Min.X
	height := bounds.Max.Y - bounds.Min.Y

	tileWidth := width / sprWidth
	tileHeight := height / sprHeight

	// spriteAt returns a sprite at the provided coordinates.
	spriteAt := func(x, y int) *ebiten.Image {
		return sheet.SubImage(image.Rect(x*tileWidth, (y+1)*tileHeight, (x+1)*tileWidth, y*tileHeight)).(*ebiten.Image)
	}

	s := make([][]*ebiten.Image, sprHeight)
	for y := 0; y < sprHeight; y++ {
		s[y] = make([]*ebiten.Image, sprWidth)
		for x := 0; x < sprWidth; x++ {
			s[y][x] = spriteAt(x, y)
		}
	}

	return s, nil
}

type SprInfo struct {
	X, Y, Width, Height int
}

// FindSubimages extracts subimages from a sprite sheet based on provided coordinates and dimensions.
// fileBytes: The byte slice containing the PNG data of the sprite sheet.
// xref: A pointer to a slice of SprInfo structs, where each struct defines the
//
//	rectangle (X, Y, Width, Height) of a subimage to extract.
//
// Returns a slice of *ebiten.Image corresponding to the defined subimages.
func LoadSubimages(fileBytes []byte, sprMap *[]SprInfo) ([]*ebiten.Image, error) {
	if sprMap == nil || len(*sprMap) == 0 {
		return []*ebiten.Image{}, nil // Return empty slice if no info provided
	}

	sheet, err := elements.LoadImage(fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	subimages := make([]*ebiten.Image, 0, len(*sprMap))

	for _, info := range *sprMap {
		// Define the rectangle for the subimage using the info from the JSON map
		rect := image.Rect(info.X, info.Y, info.X+info.Width, info.Y+info.Height)

		// Ensure the rectangle is within the bounds of the sheet
		// Note: SubImage panics if rect is outside bounds, so this check is important
		// although ideally the JSON generation script ensures valid rects.
		if !rect.In(sheet.Bounds()) {
			fmt.Printf("Warning: Subimage rectangle %v is outside the sheet bounds %v. Skipping.\n", rect, sheet.Bounds())
			continue
		}
		if rect.Empty() {
			fmt.Printf("Warning: Subimage rectangle %v is empty. Skipping.\n", rect)
			continue
		}

		// Extract the subimage
		// The result of SubImage needs type assertion to *ebiten.Image
		subImg := sheet.SubImage(rect).(*ebiten.Image)
		subimages = append(subimages, subImg)
	}

	return subimages, nil
}

// LoadSprInfoFromJSON unmarshals a JSON byte slice into a slice of SprInfo structs.
// jsonData: A byte slice containing the JSON array of subimage information.
// Returns a slice of SprInfo structs and an error if unmarshalling fails.
func LoadSprInfoFromJSON(jsonData []byte) ([]SprInfo, error) {
	var sprInfoList []SprInfo
	err := json.Unmarshal(jsonData, &sprInfoList)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}
	return sprInfoList, nil
}

// LoadMappedSprite loads a mapped sprite sheet using names and dimensions in spriteMap.
// Args:
// - spriteData - image file
// - spriteMap - a JSON file where the key is the name of the sprite and the value is the subimage details.
//
// Ex:
//
//	"btn1_norm": {
//	  "x": 1,
//	  "y": 1,
//	  "width": 93,
//	  "height": 27
//	},
//
// btn1_norm is the name of the sprite, x,y are the starting locations in the image and width/height are the width and height of the sprite image.
//
// Return
// - a map of strings to ebiten.images (subimages from the original image)
func LoadMappedSprite(spriteData []byte, spriteMap []byte) (map[string]*ebiten.Image, error) {
	var spriteInfoMap map[string]SubimageInfo
	err := json.Unmarshal(spriteMap, &spriteInfoMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal sprite map JSON: %w", err)
	}

	sheet, err := elements.LoadImage(spriteData)
	if err != nil {
		return nil, fmt.Errorf("failed to load sprite sheet image: %w", err)
	}

	sprites := make(map[string]*ebiten.Image)

	for name, info := range spriteInfoMap {
		rect := image.Rect(info.X, info.Y, info.X+info.Width, info.Y+info.Height)

		if !rect.In(sheet.Bounds()) {
			fmt.Printf("Warning: Sprite '%s' rectangle %v is outside the sheet bounds %v. Skipping.\n", name, rect, sheet.Bounds())
			continue
		}
		if rect.Empty() {
			fmt.Printf("Warning: Sprite '%s' rectangle %v is empty. Skipping.\n", name, rect)
			continue
		}

		subImg := sheet.SubImage(rect).(*ebiten.Image)
		sprites[name] = subImg
	}

	return sprites, nil
}

// LoadTTFFont parses TTF font data and returns a standard font.Face.
// fontBytes: A byte slice containing the TTF font file data.
// Returns a font.Face and an error if parsing fails.
func LoadTTFFont(fontBytes []byte) (font.Face, error) {
	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse font: %w", err)
	}

	// You can customize options like Size, DPI, Hinting here if needed.
	// Using defaults for now. Standard size is 12pt at 72 DPI.
	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    12, // Example size in points
		DPI:     72, // Standard DPI
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create font face: %w", err)
	}

	return face, nil
}
