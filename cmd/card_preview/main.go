package main

import (
	"flag"
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/hajimehoshi/ebiten/v2"
)

const previewWidth = 820
const previewHeight = 440

type previewView struct {
	name  string
	size  domain.CardView
	img   *ebiten.Image
	label *elements.Text
	x     int
}

type cardPreview struct {
	card      *domain.Card
	views     []previewView
	title     *elements.Text
	artLoaded bool
}

func newCardPreview(name string) (*cardPreview, error) {
	card := domain.FindCardByName(name)
	if card == nil {
		return nil, fmt.Errorf("unknown card %q", name)
	}
	preview := &cardPreview{
		card:  card,
		title: elements.NewText(20, card.CardName, 24, 16),
		views: []previewView{
			{name: "Full", size: domain.CardViewFull},
			{name: "Full Mini", size: domain.CardViewFullMini},
			{name: "Art Only", size: domain.CardViewArtOnly},
			{name: "Art Mini", size: domain.CardViewArtMini},
		},
	}
	x := 24
	for i := range preview.views {
		view := &preview.views[i]
		view.x = x
		view.label = elements.NewText(14, fmt.Sprintf("%s (%d x %d)", view.name, view.size.X, view.size.Y), x, 52)
		x += view.size.X + 24
	}
	if err := preview.refreshImages(); err != nil {
		return nil, err
	}
	return preview, nil
}

func (g *cardPreview) refreshImages() error {
	g.artLoaded = g.card.ImageLoaded()
	for i := range g.views {
		img, err := g.card.CardImage(g.views[i].size)
		if err != nil {
			return err
		}
		g.views[i].img = img
	}
	return nil
}

func (g *cardPreview) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if g.card.ImageLoaded() != g.artLoaded {
		return g.refreshImages()
	}
	return nil
}

func (g *cardPreview) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 45, G: 45, B: 45, A: 255})
	g.title.Draw(screen, nil, 1)
	for _, view := range g.views {
		view.label.Draw(screen, nil, 1)
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(view.x), 84)
		screen.DrawImage(view.img, opts)
	}
}

func (g *cardPreview) Layout(_, _ int) (int, int) {
	return previewWidth, previewHeight
}

func main() {
	flag.Usage = func() { fmt.Fprintln(os.Stderr, `Usage: card_preview "Card Name"`) }
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	if _, err := domain.LoadEmbeddedCardImages(); err != nil {
		log.Fatal(err)
	}
	preview, err := newCardPreview(strings.Join(flag.Args(), " "))
	if err != nil {
		log.Fatal(err)
	}
	ebiten.SetWindowSize(previewWidth, previewHeight)
	ebiten.SetWindowTitle("Card Preview: " + preview.card.CardName)
	if err := ebiten.RunGame(preview); err != nil {
		log.Fatal(err)
	}
}
