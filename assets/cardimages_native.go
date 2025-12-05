//go:build !js
// +build !js

package assets

import _ "embed"

var (
	//go:embed art/cardimages.zip
	CardImages_zip []byte
)
