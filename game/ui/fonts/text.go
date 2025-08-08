package fonts

import (
	"bytes"
	"fmt"
	"image/color"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var MtgFont = mkFont(assets.Magic_ttf)
var MtgSymFont = mkFont(assets.MagicSym_ttf)

func mkFont(fontBytes []byte) *text.GoTextFaceSource {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(fontBytes))
	if err != nil {
		panic(fmt.Errorf("failed to create font source: %w", err))
	}
	return font
}

func DrawText(screen *ebiten.Image, txt string, fontFace *text.GoTextFace, options *ebiten.DrawImageOptions) {
	R, G, B, A := color.White.RGBA()
	options.ColorScale.Scale(float32(R)/65535, float32(G)/65535, float32(B)/65535, float32(A)/65535)
	textOpts := text.DrawOptions{DrawImageOptions: *options}
	text.Draw(screen, txt, fontFace, &textOpts)
}
