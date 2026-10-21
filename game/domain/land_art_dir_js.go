//go:build js

package domain

import "io/fs"

// landArtFS reports no folder. The browser build has no local filesystem for a
// player to drop art into, so alternate land art is a native-only feature.
func landArtFS() (fs.FS, bool) {
	return nil, false
}
