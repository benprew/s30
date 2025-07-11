package fonts

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

var (
	//go:embed Magim___.ttf
	Magic_ttf []byte

	//go:embed Magis___.ttf
	MagicSym_ttf []byte
)

var MtgFont = mkMtgFont()

func mkMtgFont() *text.GoTextFaceSource {
	font, err := text.NewGoTextFaceSource(bytes.NewReader(Magic_ttf))
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
