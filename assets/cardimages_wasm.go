//go:build js
// +build js

package assets

var (
	// CardImages_zip is not embedded for wasm builds to reduce binary size.
	// It will be downloaded at runtime.
	CardImages_zip []byte
)
