//go:build embedded_card_images

package assets

import _ "embed"

// CardImagesZip contains the card artwork included in standalone builds.
//
//go:embed art/cardimages.zip
var CardImagesZip []byte
