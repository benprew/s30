package domain

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func TestClassifyLandArt(t *testing.T) {
	tests := []struct {
		filename string
		want     string
		wantOK   bool
	}{
		{"swamp.png", "swamp", true},
		{"Forest.jpg", "forest", true},
		{"ISLAND.JPEG", "island", true},
		{"Island.full.jpg", "island", true},
		{"mountain-beta-3.PNG", "mountain", true},
		{"plains4.png", "plains", true},
		{"beta/forest1.png", "forest", true},
		{"notes.txt", "", false},
		{"swamp.gif", "", false},
		{"forest.psd", "", false},
		{"treasure.png", "", false},
		{"swamp", "", false},
		{"", "", false},
		{".png", "", false},
		{"._forest.png", "", false},
	}

	for _, tt := range tests {
		got, ok := classifyLandArt(tt.filename)
		if ok != tt.wantOK {
			t.Errorf("classifyLandArt(%q) ok = %v, want %v", tt.filename, ok, tt.wantOK)
			continue
		}
		if got != tt.want {
			t.Errorf("classifyLandArt(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

func TestClassifyLandArtCoversEveryBasicLand(t *testing.T) {
	for name := range basicLands {
		got, ok := classifyLandArt(name + ".png")
		if !ok {
			t.Errorf("classifyLandArt(%q) found no land, want a match", name+".png")
			continue
		}
		if card := FindCardByName(name); card == nil {
			t.Errorf("basic land %q is not in CARDS", name)
		}
		if got == "" {
			t.Errorf("classifyLandArt(%q) returned an empty land name", name+".png")
		}
	}
}

func TestScanLandArtFlatFolder(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"swamp1.png":     &fstest.MapFile{},
		"swamp2.jpg":     &fstest.MapFile{},
		"Swamp3.webp":    &fstest.MapFile{},
		"readme.txt":     &fstest.MapFile{},
	})

	want := []string{"swamp1.png", "swamp2.jpg"}
	if got := lib.Paths("swamp"); !reflect.DeepEqual(got, want) {
		t.Errorf("Paths(\"swamp\") = %v, want %v", got, want)
	}
	if got := lib.Total(); got != 2 {
		t.Errorf("Total() = %d, want 2", got)
	}
}

func TestScanLandArtNestedFolders(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"beta/Forest.full.jpg":      &fstest.MapFile{},
		"beta/Island.full.jpg":      &fstest.MapFile{},
		"unlimited/Forest.full.jpg": &fstest.MapFile{},
		"forest-loose.png":          &fstest.MapFile{},
		"beta/notes.txt":            &fstest.MapFile{},
	})

	wantForest := []string{"beta/Forest.full.jpg", "forest-loose.png", "unlimited/Forest.full.jpg"}
	if got := lib.Paths("forest"); !reflect.DeepEqual(got, wantForest) {
		t.Errorf("Paths(\"forest\") = %v, want %v", got, wantForest)
	}
	if got := lib.Paths("island"); len(got) != 1 {
		t.Errorf("Paths(\"island\") = %v, want 1 entry", got)
	}
	if got := lib.Total(); got != 4 {
		t.Errorf("Total() = %d, want 4", got)
	}
}

func TestScanLandArtIgnoresDirectoryNames(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{
		"swamp.png/placeholder.txt": &fstest.MapFile{},
		"forest/island.jpg":         &fstest.MapFile{},
	})

	if got := lib.Paths("swamp"); len(got) != 0 {
		t.Errorf("a directory named swamp.png was collected as art: %v", got)
	}
	if got := lib.Paths("forest"); len(got) != 0 {
		t.Errorf("a directory named forest was collected as art: %v", got)
	}
	if got := lib.Paths("island"); len(got) != 1 {
		t.Errorf("Paths(\"island\") = %v, want the file inside the forest folder", got)
	}
}

func TestPathsAcceptsCardName(t *testing.T) {
	lib := ScanLandArt(fstest.MapFS{"Mountain.full.png": &fstest.MapFile{}})

	if got := lib.Paths("Mountain"); len(got) != 1 {
		t.Errorf("Paths(\"Mountain\") = %v, want 1 entry", got)
	}
}

func TestScanLandArtEmptyAndNil(t *testing.T) {
	empty := ScanLandArt(fstest.MapFS{})
	if empty == nil {
		t.Fatal("ScanLandArt returned nil for an empty filesystem")
	}
	if got := empty.Total(); got != 0 {
		t.Errorf("Total() = %d, want 0", got)
	}

	if got := ScanLandArt(nil).Total(); got != 0 {
		t.Errorf("ScanLandArt(nil).Total() = %d, want 0", got)
	}

	var missing *LandArtLibrary
	if got := missing.Paths("forest"); got != nil {
		t.Errorf("(*LandArtLibrary)(nil).Paths() = %v, want nil", got)
	}
	if got := missing.Total(); got != 0 {
		t.Errorf("(*LandArtLibrary)(nil).Total() = %d, want 0", got)
	}
}
