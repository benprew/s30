package duel

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"testing"
	"testing/fstest"

	"github.com/benprew/mage-go/pkg/mage/interactive"
	"github.com/benprew/s30/game/domain"
	"github.com/google/uuid"
)

func TestPermanentArtName_ChangedBasicLandSubtype(t *testing.T) {
	cityOfBrass := &domain.Card{
		CardName: "City of Brass",
		CardType: domain.CardTypeLand,
	}
	perm := interactive.PermanentState{
		Name:     "City of Brass",
		IsLand:   true,
		SubTypes: "Forest",
	}

	if got := permanentArtName(perm, cityOfBrass); got != "Forest" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest")
	}
}

func TestPermanentArtName_UnchangedBasicLandSubtype(t *testing.T) {
	forest := &domain.Card{
		CardName: "Forest",
		CardType: domain.CardTypeLand,
		Subtypes: []string{"Forest"},
	}
	perm := interactive.PermanentState{
		Name:     "Forest",
		IsLand:   true,
		SubTypes: "Forest",
	}

	if got := permanentArtName(perm, forest); got != "Forest" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest")
	}
}

func TestPermanentArtName_AddedBasicLandSubtype(t *testing.T) {
	forest := &domain.Card{
		CardName: "Forest",
		CardType: domain.CardTypeLand,
		Subtypes: []string{"Forest"},
	}
	perm := interactive.PermanentState{
		Name:     "Forest",
		IsLand:   true,
		SubTypes: "Forest Mountain",
	}

	if got := permanentArtName(perm, forest); got != "Mountain" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Mountain")
	}
}

func TestPermanentArtName_RevertsWhenSubtypeChangeEnds(t *testing.T) {
	cityOfBrass := &domain.Card{
		CardName: "City of Brass",
		CardType: domain.CardTypeLand,
	}
	perm := interactive.PermanentState{
		Name:   "City of Brass",
		IsLand: true,
	}

	if got := permanentArtName(perm, cityOfBrass); got != "City of Brass" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "City of Brass")
	}
}

func TestPermanentArtName_NonlandWithSubtypeWord(t *testing.T) {
	card := &domain.Card{CardName: "Forest Bear", CardType: domain.CardTypeCreature}
	perm := interactive.PermanentState{
		Name:     "Forest Bear",
		SubTypes: "Bear Forest",
	}

	if got := permanentArtName(perm, card); got != "Forest Bear" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Forest Bear")
	}
}

func TestBuildCardImageMap_IncludesBasicLands(t *testing.T) {
	cardImages := buildCardImageMap(nil)
	for _, name := range basicLandNames {
		if cardImages[name] == nil {
			t.Errorf("buildCardImageMap() missing %q", name)
		}
	}
}

// A dual land has printed basic land types, so nothing about it has changed and
// it must keep its own art instead of degrading to one of the basics.
func TestPermanentArtName_DualLandKeepsItsOwnArt(t *testing.T) {
	duals := map[string]string{
		"Badlands":        "Swamp Mountain",
		"Bayou":           "Swamp Forest",
		"Plateau":         "Mountain Plains",
		"Savannah":        "Forest Plains",
		"Scrubland":       "Plains Swamp",
		"Taiga":           "Mountain Forest",
		"Tropical Island": "Forest Island",
		"Tundra":          "Plains Island",
		"Underground Sea": "Island Swamp",
		"Volcanic Island": "Island Mountain",
	}

	for name, subtypes := range duals {
		printed := domain.FindCardByName(name)
		if printed == nil {
			t.Errorf("%s missing from the card database", name)
			continue
		}
		perm := interactive.PermanentState{Name: name, IsLand: true, SubTypes: subtypes}
		if got := permanentArtName(perm, printed); got != name {
			t.Errorf("permanentArtName(%s) = %q, want %q", name, got, name)
		}
	}
}

// Blood Moon-style effects still override a dual land's art.
func TestPermanentArtName_DualLandTurnedIntoMountain(t *testing.T) {
	printed := domain.FindCardByName("Underground Sea")
	if printed == nil {
		t.Fatal("Underground Sea missing from the card database")
	}
	perm := interactive.PermanentState{Name: "Underground Sea", IsLand: true, SubTypes: "Mountain"}

	if got := permanentArtName(perm, printed); got != "Mountain" {
		t.Fatalf("permanentArtName() = %q, want %q", got, "Mountain")
	}
}

func landArtFixture(t *testing.T, names ...string) *domain.LandArtLibrary {
	t.Helper()
	files := fstest.MapFS{}
	for i, name := range names {
		img := image.NewRGBA(image.Rect(0, 0, 490, 684))
		img.Set(0, 0, color.RGBA{R: uint8(i + 1), A: 0xff})
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			t.Fatalf("failed to encode %s: %v", name, err)
		}
		files[name] = &fstest.MapFile{Data: buf.Bytes()}
	}
	return domain.ScanLandArt(files)
}

func landArtDuelScreen(t *testing.T, lib *domain.LandArtLibrary, seed int64) (*DuelScreen, *domain.Card) {
	t.Helper()
	// Real cards, with their art pre-cached so nothing reaches for the network.
	forest := domain.FindCardByName("Forest")
	bolt := domain.FindCardByName("Lightning Bolt")
	if forest == nil || bolt == nil {
		t.Fatal("Forest or Lightning Bolt is missing from the card database")
	}
	for _, c := range []*domain.Card{forest, bolt} {
		domain.CacheCardImage(c.CardID(), image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, 342)))
	}

	var sel *domain.LandArtSelection
	if lib != nil {
		sel = domain.NewLandArtSelectionFor(lib, rand.New(rand.NewSource(seed)))
	}
	return &DuelScreen{
		cardImageMap: map[string]*domain.Card{"Forest": forest, "Lightning Bolt": bolt},
		landArt:      sel,
	}, forest
}

func forestPermanent(id uuid.UUID) interactive.PermanentState {
	return interactive.PermanentState{ID: id, Name: "Forest", IsLand: true, SubTypes: "Forest"}
}

// CardViewFull hands back the source image itself, so these comparisons are
// over the art file each draw actually used.
func TestDuelPreviewShowsTheSameLandArtAsTheBattlefield(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	lib := landArtFixture(t, "forest1.png", "forest2.png", "forest3.png", "forest4.png")
	s, forest := landArtDuelScreen(t, lib, 7)

	permID := uuid.New()
	perm := forestPermanent(permID)
	battlefield := s.getCardArtImg("Forest", permID.String(), domain.CardViewFull)
	if battlefield == nil {
		t.Fatal("battlefield draw produced no art")
	}

	s.preview = &cardPreview{Card: forest, Permanent: &perm}
	if got := s.previewImage(); got != battlefield {
		t.Error("the magnified preview shows different land art than the battlefield card")
	}

	// Other permanents roll independently. Rolls come from a fixed seed, so this
	// is deterministic, but a single comparison would be hostage to that seed:
	// seed 7 happens to give the first other permanent the same art as the
	// battlefield. Sampling several keeps the check meaningful if the seed or
	// the fixture changes.
	sawDifferent := false
	for range 5 {
		otherPerm := forestPermanent(uuid.New())
		s.preview = &cardPreview{Card: forest, Permanent: &otherPerm}
		if s.previewImage() != battlefield {
			sawDifferent = true
			break
		}
	}
	if !sawDifferent {
		t.Error("every permanent previewed the first permanent's art, so the key is being ignored")
	}
}

func TestDuelLandArtIsStableForOneInstance(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	s, _ := landArtDuelScreen(t, landArtFixture(t, "forest1.png", "forest2.png", "forest3.png"), 3)

	key := uuid.New().String()
	first := s.getCardArtImg("Forest", key, domain.CardViewFull)
	second := s.getCardArtImg("Forest", key, domain.CardViewFull)

	if first == nil {
		t.Fatal("no art was drawn")
	}
	if first != second {
		t.Error("one land was given different art on a second draw")
	}
}

func TestDuelWithoutLandArtDrawsPrintedArt(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	s, forest := landArtDuelScreen(t, nil, 1)

	want, err := forest.CardImage(domain.CardViewArtMini)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	if got := s.getCardArtImg("Forest", uuid.New().String(), domain.CardViewArtMini); got != want {
		t.Error("with no land art installed, a Forest did not draw its printed art")
	}
}

func TestDuelLandArtLeavesNonLandsAlone(t *testing.T) {
	t.Cleanup(domain.ClearCardImageCache)
	s, _ := landArtDuelScreen(t, landArtFixture(t, "forest1.png", "forest2.png"), 1)

	bolt := s.cardImageMap["Lightning Bolt"]
	want, err := bolt.CardImage(domain.CardViewArtOnly)
	if err != nil {
		t.Fatalf("CardImage returned an error: %v", err)
	}
	if got := s.getCardArtImg("Lightning Bolt", uuid.New().String(), domain.CardViewArtOnly); got != want {
		t.Error("a non-land card did not draw its printed art")
	}
}
