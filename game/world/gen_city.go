package world

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/hajimehoshi/ebiten/v2"
)

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
	var err error
	if loadedCityImage == nil {
		// Load city image
		loadedCityImage, err = imageutil.LoadImage(assets.City_png)
		if err != nil {
			panic(fmt.Sprintf("Unable to load cityBgImage: %s", err))
		}
	}
	if loadedVillageImage == nil {
		// Load village image
		loadedVillageImage, err = imageutil.LoadImage(assets.Village_png)
		if err != nil {
			panic(fmt.Sprintf("Unable to load villageBgImage: %s", err))
		}
	}

	// Return the appropriate image based on tier
	if tier == 1 {
		return loadedVillageImage
	}
	return loadedCityImage
}
