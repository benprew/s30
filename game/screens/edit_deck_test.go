package screens

import (
	"bytes"
	"image"
	"slices"
	"testing"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/screenui"
)

func TestLoadEditDeckBackgroundUsesBlueManaTerrain(t *testing.T) {
	background, err := loadEditDeckBackground(image.Rect(0, 0, 1024, 768))
	if err != nil {
		t.Fatalf("loadEditDeckBackground() error = %v", err)
	}

	if got, want := background.Bounds(), image.Rect(0, 0, 1024, 768); got != want {
		t.Fatalf("background bounds = %v, want %v", got, want)
	}

	source, _, err := image.Decode(bytes.NewReader(assets.EditDeckTerrain_png))
	if err != nil {
		t.Fatalf("decode blue mana terrain: %v", err)
	}
	if got, want := source.Bounds(), image.Rect(0, 0, 721, 381); got != want {
		t.Fatalf("blue mana terrain bounds = %v, want %v", got, want)
	}
}

// The exit button shares the filter row's band, at its right end, which is where the
// original keeps its Done. Pinned so a later layout change cannot quietly move it back
// to the top corner or on top of the toggles.
func TestEditDeckExitButtonSharesTheFilterRowBand(t *testing.T) {
	const W, H = 1024, 768
	exit := editDeckBackBounds(W)
	bandTop := H - COLLECTION_HEIGHT - filterBtnSize - filterGapAboveCarousel
	bandBottom := bandTop + filterBtnSize

	if exit.Min.Y < bandTop || exit.Max.Y > bandBottom {
		t.Errorf("exit button %v is not inside the filter row's band %d..%d", exit, bandTop, bandBottom)
	}
	if exit.Max.X != W-12 {
		t.Errorf("exit button right edge = %d, want %d", exit.Max.X, W-12)
	}
	// The toggles grow from the left edge of the row, so the exit button has to stay
	// clear of the space they occupy, with room left for the ones still to come.
	if exit.Min.X < 500 {
		t.Errorf("exit button starts at x=%d, which the filter toggles can reach", exit.Min.X)
	}
}

func TestEditDeckSellBoundsAtTopOfScreen(t *testing.T) {
	want := image.Rect(10, 12, 150, 68)

	if got := editDeckSellBounds(); got != want {
		t.Fatalf("editDeckSellBounds() = %v, want %v", got, want)
	}
}

func TestEditDeckSortsDeckAndCollectionSeparately(t *testing.T) {
	wantDeck := []string{
		"Healing Salve", "Savannah Lions", "Disenchant",
		"Counterspell", "Air Elemental",
		"Dark Ritual", "Terror",
		"Disintegrate", "Fireball", "Lightning Bolt", "Shivan Dragon",
		"Giant Growth",
		"Dark Heart of the Wood", "Marsh Goblins",
		"Black Lotus", "Plains", "Sol Ring",
	}
	collection := domain.NewCardCollection()
	art := image.NewRGBA(image.Rect(0, 0, domain.CardFullWidth, domain.CardFullHeight))
	t.Cleanup(domain.ClearCardImageCache)
	for _, name := range slices.Backward(wantDeck) {
		printing := domain.FindCardByName(name).Printing("")
		domain.CacheCardImage(printing.ArtworkID(), art)
		collection.AddPrinting(printing, 3)
		if err := collection.MovePrintingToDeck(printing, 0, 2); err != nil {
			t.Fatal(err)
		}
	}
	player := &domain.Player{Character: domain.Character{CardCollection: collection}}
	screen, err := NewEditDeckScreen(player, nil, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}

	var deckNames []string
	for _, card := range screen.deckCardDisplays {
		deckNames = append(deckNames, card.owned.Name())
		data := card.drag.StartDrag(0, 0).GetData().(deckCardDrag)
		if card.count != 2 || data.printing != card.owned || !data.fromDeck {
			t.Errorf("incorrect count or drag target for %s", card.owned.Name())
		}
	}
	if !slices.Equal(deckNames, wantDeck) {
		t.Errorf("deck order = %v, want %v", deckNames, wantDeck)
	}
	wantCollection := slices.Clone(wantDeck)
	slices.Sort(wantCollection)
	var collectionNames []string
	for _, button := range screen.CollectionList.GetItems() {
		collectionNames = append(collectionNames, button.ID)
	}
	if !slices.Equal(collectionNames, wantCollection) {
		t.Errorf("collection order = %v, want %v", collectionNames, wantCollection)
	}
}

func TestSellDroppedCardFromCollection(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	printing := mountain.Printing("")
	collection.AddPrinting(printing, 2)
	for slot, count := range []int{1, 2} {
		if err := collection.MovePrintingToDeck(printing, slot, count); err != nil {
			t.Fatal(err)
		}
	}
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}

	dropped := screen.handleCardDropToSell(firstCollectionDrag(screen))

	if !dropped {
		t.Fatal("collection card sale failed")
	}
	if got := collection.GetTotalCount(mountain); got != 1 {
		t.Fatalf("collection count = %d, want 1", got)
	}
	for slot, want := range [domain.MaxPlayerDecks]int{1, 1, 0} {
		if got := collection.GetDeckCount(mountain, slot); got != want {
			t.Errorf("deck %d count after selling from collection = %d, want %d", slot, got, want)
		}
	}
}

func TestSellDroppedCardFromDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	printing := mountain.Printing("")
	collection.AddPrinting(printing, 2)
	for slot, count := range []int{1, 1, 2} {
		if err := collection.MovePrintingToDeck(printing, slot, count); err != nil {
			t.Fatal(err)
		}
	}
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}

	data := screen.deckCardDisplays[0].drag.StartDrag(0, 0)
	if !screen.handleCardDropToSell(data) {
		t.Fatal("deck card sale failed")
	}
	if got := collection.GetTotalCount(mountain); got != 1 {
		t.Fatalf("collection count = %d, want 1", got)
	}
	for slot, want := range [domain.MaxPlayerDecks]int{0, 1, 1} {
		if got := collection.GetDeckCount(mountain, slot); got != want {
			t.Errorf("deck %d count after selling from deck = %d, want %d", slot, got, want)
		}
	}
	if screen.handleCardDropToSell(data) || player.Gold != mountain.SalePrice(screen.City) {
		t.Fatalf("gold after repeated drop = %d, want one sale", player.Gold)
	}
}

func firstCollectionDrag(screen *EditDeckScreen) dragdrop.DragData {
	button := screen.CollectionList.GetItems()[0]
	return screen.draggableItems[button].StartDrag(0, 0)
}

func TestCreateCollectionButtons_ExcludesDeckCards(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	forest := domain.FindCardByName("Forest")
	plains := domain.FindCardByName("Plains")

	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 8, 0)
	collection.AddCard(mountain, 5, -1)
	collection.AddCard(forest, 5, 0)
	collection.AddCard(plains, 3, -1)

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}

	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	var mountainGroup *cardGroup
	var forestGroup *cardGroup
	var plainsGroup *cardGroup

	for name, group := range screen.collectionGroups {
		switch name {
		case "Mountain":
			mountainGroup = group
		case "Forest":
			forestGroup = group
		case "Plains":
			plainsGroup = group
		}
	}

	if mountainGroup == nil {
		t.Error("Mountain should appear in collection list (13 total, 8 in deck, 5 available)")
	} else if mountainGroup.totalCount != 5 {
		t.Errorf("Mountain available count should be 5, got %d", mountainGroup.totalCount)
	}

	if forestGroup != nil {
		t.Error("Forest should NOT appear in collection list (5 total, 5 in deck, 0 available)")
	}

	if plainsGroup == nil {
		t.Error("Plains should appear in collection list (3 total, 0 in deck, 3 available)")
	} else if plainsGroup.totalCount != 3 {
		t.Errorf("Plains available count should be 3, got %d", plainsGroup.totalCount)
	}

	expectedButtonCount := 2
	if len(buttons) != expectedButtonCount {
		t.Errorf("Expected %d buttons (Mountain and Plains), got %d", expectedButtonCount, len(buttons))
	}
}

func TestCreateCollectionButtons_RemovesWhenAllInDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")

	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 10, 0)

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}
	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	if len(buttons) != 0 {
		t.Errorf("Expected 0 buttons when all 10 cards are in deck, got %d buttons", len(buttons))
	}

	mountainGroup := screen.collectionGroups["Mountain"]
	if mountainGroup != nil {
		t.Error("Mountain should NOT appear in collection list when all copies are in deck")
	}
}

func TestMovePrintingFromDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	printing := mountain.Printing("")

	collection := domain.NewCardCollection()
	collection.AddPrinting(printing, 10).DeckCounts = []int{5}

	totalCount := collection.GetTotalCount(mountain)
	deckCount := collection.GetDeckCount(mountain, 0)

	if totalCount != 10 {
		t.Errorf("Expected total count to be 10, got %d", totalCount)
	}
	if deckCount != 5 {
		t.Errorf("Expected deck count to be 5, got %d", deckCount)
	}

	err := collection.MovePrintingFromDeck(printing, 0, 2)
	if err != nil {
		t.Fatalf("Failed to move card from deck: %v", err)
	}

	totalCount = collection.GetTotalCount(mountain)
	deckCount = collection.GetDeckCount(mountain, 0)

	if totalCount != 10 {
		t.Errorf("Expected total count to remain 10, got %d", totalCount)
	}
	if deckCount != 3 {
		t.Errorf("Expected deck count to be 3 after removing 2, got %d", deckCount)
	}

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: collection,
		},
		ActiveDeck: 0,
	}

	screen := &EditDeckScreen{
		Player:           player,
		collectionGroups: make(map[string]*cardGroup),
	}

	buttons, err := screen.createCollectionButtons()
	if err != nil {
		t.Fatalf("Failed to create collection buttons: %v", err)
	}

	if len(buttons) != 1 {
		t.Errorf("Expected 1 button, got %d", len(buttons))
	}

	mountainGroup := screen.collectionGroups["Mountain"]
	if mountainGroup == nil {
		t.Fatal("Mountain should appear in collection list")
		return
	}
	if mountainGroup.totalCount != 7 {
		t.Errorf("Expected 7 available (10 total - 3 in deck), got %d", mountainGroup.totalCount)
	}
}

func TestHandleCardDrop_RejectsCardFromDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 1, 0)
	collection.AddCard(mountain, 2, -1)
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatalf("NewEditDeckScreen() error = %v", err)
	}

	deckCardData := screen.deckCardDisplays[0].drag.StartDrag(0, 0)
	dropped := screen.handleCardDrop(deckCardData)
	if dropped {
		t.Fatal("handleCardDrop() = true, want false when card dragged from deck")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 1 {
		t.Fatalf("deck count = %d, want 1", got)
	}
	if got := collection.GetTotalCount(mountain); got != 3 {
		t.Fatalf("collection count = %d, want 3", got)
	}
}

func TestHandleCardDropToCollection_RejectsCardFromCollection(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 1, 0)
	collection.AddCard(mountain, 2, -1)
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatalf("NewEditDeckScreen() error = %v", err)
	}

	collectionCardData := firstCollectionDrag(screen)
	dropped := screen.handleCardDropToCollection(collectionCardData)
	if dropped {
		t.Fatal("handleCardDropToCollection() = true, want false when card dragged from collection")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 1 {
		t.Fatalf("deck count = %d, want 1", got)
	}
}

func TestEditDeckDropAreaAcceptanceRules(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 1, 0)
	collection.AddCard(mountain, 1, -1)
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatalf("NewEditDeckScreen() error = %v", err)
	}

	deckCardData := screen.deckCardDisplays[0].drag.StartDrag(0, 0)
	collectionCardData := firstCollectionDrag(screen)

	if screen.deckDropArea.CanAcceptDrop(deckCardData) {
		t.Error("deckDropArea should not accept card dragged from deck")
	}
	if !screen.deckDropArea.CanAcceptDrop(collectionCardData) {
		t.Error("deckDropArea should accept card dragged from collection")
	}

	if !screen.collectionDropArea.CanAcceptDrop(deckCardData) {
		t.Error("collectionDropArea should accept card dragged from deck")
	}
	if screen.collectionDropArea.CanAcceptDrop(collectionCardData) {
		t.Error("collectionDropArea should not accept card dragged from collection")
	}
}

func TestAddDroppedCardToDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 1, 0)
	collection.AddCard(mountain, 2, -1)
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatalf("NewEditDeckScreen() error = %v", err)
	}

	deckCardData := screen.deckCardDisplays[0].drag.StartDrag(0, 0)
	if screen.handleCardDrop(deckCardData) {
		t.Fatal("handleCardDrop() = true, want false when card dragged from deck")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 1 {
		t.Fatalf("deck count = %d, want 1", got)
	}

	collectionCardData := firstCollectionDrag(screen)
	if !screen.handleCardDrop(collectionCardData) {
		t.Fatal("handleCardDrop() = false, want true when card dragged from collection")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 2 {
		t.Fatalf("deck count = %d, want 2", got)
	}
}

func TestRemoveDroppedCardFromDeck(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	collection := domain.NewCardCollection()
	collection.AddCard(mountain, 1, 0)
	collection.AddCard(mountain, 2, -1)
	player := &domain.Player{
		Character:  domain.Character{CardCollection: collection},
		ActiveDeck: 0,
	}
	screen, err := NewEditDeckScreen(player, &domain.City{Tier: domain.TierHamlet}, 1024, 768)
	if err != nil {
		t.Fatalf("NewEditDeckScreen() error = %v", err)
	}

	collectionCardData := firstCollectionDrag(screen)
	if screen.handleCardDropToCollection(collectionCardData) {
		t.Fatal("handleCardDropToCollection() = true, want false when card dragged from collection")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 1 {
		t.Fatalf("deck count = %d, want 1", got)
	}

	deckCardData := screen.deckCardDisplays[0].drag.StartDrag(0, 0)
	if !screen.handleCardDropToCollection(deckCardData) {
		t.Fatal("handleCardDropToCollection() = false, want true when card dragged from deck")
	}
	if got := collection.GetDeckCount(mountain, 0); got != 0 {
		t.Fatalf("deck count = %d, want 0", got)
	}
}

// The deck editor is where a deck gets finished, so the player should learn there
// that it is short, instead of finding out in a duel that silently pads it.
func TestLeavingWithAShortDeckWarnsInsteadOfClosing(t *testing.T) {
	player := testPlayerWithDeck(t, 3)
	screen := &EditDeckScreen{Player: player, City: &domain.City{Tier: domain.TierHamlet}}

	if got := screen.leaveEditor(); got != screenui.EditDeckScr {
		t.Fatalf("leaveEditor() = %v, want the editor to stay open", got)
	}
	if !screen.leaveWarning {
		t.Error("leaveWarning = false, want a warning when the deck is below the minimum")
	}
	if screen.leaveWarningBody == "" {
		t.Error("leaveWarningBody is empty, want the deck size and the minimum in the warning")
	}
}

func TestLeavingWithALegalDeckCloses(t *testing.T) {
	player := testPlayerWithDeck(t, 36)
	screen := &EditDeckScreen{Player: player, City: &domain.City{Tier: domain.TierHamlet}}

	if got := screen.leaveEditor(); got != screenui.CityScr {
		t.Fatalf("leaveEditor() = %v, want %v", got, screenui.CityScr)
	}
	if screen.leaveWarning {
		t.Error("leaveWarning = true, want no warning for a legal deck")
	}
}

func TestDeckCardsDoNotOverflowDeckArea(t *testing.T) {
	screen := newEditorWithDistinctDeck(t, 42)
	bounds := screen.deckDropArea.GetDropBounds()
	visible := screen.visibleDeckCards()
	if len(visible) != 42 {
		t.Fatalf("visible cards = %d, want 42", len(visible))
	}

	for i, display := range visible {
		cardBounds := display.drag.Bounds
		wantY := 55 + i/6*64
		if cardBounds.Min.Y != wantY {
			t.Errorf("card %d starts at y=%d, want %d", i, cardBounds.Min.Y, wantY)
		}
		cardImageBounds := image.Rectangle{Min: cardBounds.Min, Max: cardBounds.Min.Add(display.drag.Normal.Bounds().Size())}
		if !cardImageBounds.In(bounds) {
			t.Errorf("card %s at %v extends outside deck area %v", display.owned.Name(), cardImageBounds, bounds)
		}
	}
}

func TestVisibleDraggablesFollowDeckPages(t *testing.T) {
	screen := newEditorWithDistinctDeck(t, 121)
	first := screen.deckCardDisplays[0].owned
	last := screen.deckCardDisplays[120].owned
	if got := len(screen.visibleDraggables()); got != 120 {
		t.Fatalf("first page has %d drag targets, want 120", got)
	}
	if !screen.setDeckPage(1) {
		t.Fatal("second page did not open")
	}
	targets := screen.visibleDraggables()
	if len(targets) != 1 {
		t.Fatalf("second page has %d drag targets, want 1", len(targets))
	}
	start := image.Pt(320, 60)
	drag := targets[0].StartDrag(start.X, start.Y).GetData().(deckCardDrag)
	if got, fromDeck := screen.cardAt(start); got != last || !fromDeck || drag.printing != last {
		t.Fatal("second page hover and drag did not select its only card")
	}

	if !screen.removeDeckCard(last) {
		t.Fatal("could not remove the last card on the second page")
	}
	if got, fromDeck := screen.cardAt(start); screen.deckPage != 0 || got != first || !fromDeck {
		t.Fatal("removing the last card on the second page did not return to the first page")
	}
}

func newEditorWithDistinctDeck(t *testing.T, count int) *EditDeckScreen {
	t.Helper()
	collection := domain.NewCardCollection()
	names := make(map[string]bool)
	for _, card := range domain.CARDS {
		if names[card.Name()] {
			continue
		}
		names[card.Name()] = true
		collection.AddCard(card, 1, 0)
		if len(names) == count {
			break
		}
	}
	if len(names) != count {
		t.Fatalf("could not find %d distinct cards", count)
	}
	player := &domain.Player{Character: domain.Character{CardCollection: collection}}
	screen, err := NewEditDeckScreen(player, nil, 1024, 768)
	if err != nil {
		t.Fatal(err)
	}
	return screen
}

func testPlayerWithDeck(t *testing.T, cards int) *domain.Player {
	t.Helper()
	plains := domain.FindCardByName("Plains")
	collection := domain.NewCardCollection()
	collection.AddCard(plains, cards, 0)
	return &domain.Player{
		Character:   domain.Character{CardCollection: collection},
		MinDeckSize: 36,
		ActiveDeck:  0,
	}
}
