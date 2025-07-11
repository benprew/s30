package world

import (
	"bytes"
	"fmt"
	"image"
	"math/rand"
	"strings"

	"github.com/benprew/s30/assets/art"
	"github.com/hajimehoshi/ebiten/v2"
)

// represents the cities and villages on the map
type City struct {
	tier            int // 1,2 or 3
	Name            string
	X               int
	Y               int
	population      int
	BackgroundImage *ebiten.Image
}

var cityPrefixes = []string{"Sharmal",
	"Ardestan",
	"Azar",
	"Bakur",
	"Freyalise",
	"Jalira",
	"Kurkesh",
	"Talrand",
	"Yisan",
}

var cityPostfixes = []string{
	"Bastion",
	"Citadel",
	"Crag",
	"Crypt",
	"Den",
	"Fane",
	"Fortress",
	"Glade",
	"Grove",
	"Hallow",
	"Haven",
	"Hold",
	"Keep",
	"March",
	"Mere",
	"Oasis",
	"Pillar",
	"Refuge",
	"Sanctum",
	"Shrine",
	"Spire",
	"Steading",
	"Temple",
	"Thorne",
	"Tower",
	"Vance",
	"Ward",
	"Wold",
}

var (
	loadedCityImage    *ebiten.Image
	loadedVillageImage *ebiten.Image
	cityImageLoadError error
)

func genCityName() string {
	// Pick a random prefix and make it possessive
	prefix := cityPrefixes[rand.Intn(len(cityPrefixes))]
	if !strings.HasSuffix(prefix, "s") {
		prefix += "'s"
	}

	// Pick a random postfix
	postfix := cityPostfixes[rand.Intn(len(cityPostfixes))]

	return prefix + " " + postfix
}

func cityBgImage(tier int) *ebiten.Image {
	// Check if images are already loaded
	if loadedCityImage == nil && loadedVillageImage == nil && cityImageLoadError == nil {
		// Load city image
		cityImg, _, err := image.Decode(bytes.NewReader(art.City_png))
		if err != nil {
			panic(fmt.Sprintf("Unable to load cityBgImage: %s", err))
		}
		loadedCityImage = ebiten.NewImageFromImage(cityImg)

		// Load village image
		villageImg, _, err := image.Decode(bytes.NewReader(art.Village_png))
		if err != nil {
			panic(fmt.Sprintf("Unable to load villageBgImage: %s", err))
		}
		loadedVillageImage = ebiten.NewImageFromImage(villageImg)
	}

	// Return the appropriate image based on tier
	if tier == 1 {
		return loadedVillageImage
	}
	return loadedCityImage
}

func (c *City) FoodCost() int {
	return c.tier * 10
}
