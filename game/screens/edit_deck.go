package screens

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	COLLECTION_WIDTH  = 1024
	COLLECTION_HEIGHT = 180
)

type cardGroup struct {
	name       string
	cards      []*domain.Card // All printings of this card
	totalCount int            // Total count across all printings
}

// EditDeckScreen allows players to edit their decks
type EditDeckScreen struct {
	Player           *domain.Player
	CollectionList   *elements.ScrollableList
	Background       *ebiten.Image
	DeckButtons      []*elements.Button // Buttons for cards currently in the deck
	lastClickTime    map[int]int        // Track click times for double-click detection
	clickFrame       int                // Current frame counter for double-click timing
	MagnifierImage   *ebiten.Image      // Image to display in the magnifier
	dragManager      *dragdrop.DragManager
	deckDropArea     *dragdrop.DropArea
	draggableItems   []*dragdrop.DraggableButton
	deckCardDisplays []DeckCardDisplay        // Cards currently in the deck with counts
	deckCardImages   map[string]*ebiten.Image // Cached resized card images for deck display
	collectionGroups map[string]*cardGroup    // Map card name to group
}

type DeckCardDisplay struct {
	Card  *domain.Card
	Count int
	Image *ebiten.Image
	X     int
	Y     int
}

func (s *EditDeckScreen) IsFramed() bool {
	return false
}

// NewEditDeckScreen creates a new edit deck screen
func NewEditDeckScreen(player *domain.Player, W, H int) (*EditDeckScreen, error) {
	// Load the collection background
	collectionBg, err := imageutil.LoadImage(assets.EditDeckBg_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load edit deck background: %w", err)
	}

	screen := &EditDeckScreen{
		Player:           player,
		Background:       collectionBg,
		DeckButtons:      make([]*elements.Button, 0),
		lastClickTime:    make(map[int]int),
		clickFrame:       0,
		dragManager:      dragdrop.NewDragManager(),
		draggableItems:   make([]*dragdrop.DraggableButton, 0),
		deckCardDisplays: make([]DeckCardDisplay, 0),
		deckCardImages:   make(map[string]*ebiten.Image),
		collectionGroups: make(map[string]*cardGroup),
	}

	// Create the scrollable list for the card collection
	collectionButtons, err := screen.createCollectionButtons()
	if err != nil {
		return nil, fmt.Errorf("failed to create collection buttons: %w", err)
	}

	// Create ScrollableList at the bottom of the screen
	// Use layout anchor to position it at the bottom center
	listPos := &layout.Position{
		Anchor:  layout.BottomCenter,
		OffsetX: -COLLECTION_WIDTH / 2, // Center horizontally
		OffsetY: -COLLECTION_HEIGHT,    // Position up from bottom
	}

	scrollList, err := elements.NewScrollableList(
		collectionButtons,
		collectionBg,
		COLLECTION_WIDTH,
		COLLECTION_HEIGHT,
		elements.OrientationHorizontal,
		listPos,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create scrollable list: %w", err)
	}

	screen.CollectionList = scrollList

	// Create deck area with fixed bounds
	deckAreaBounds := image.Rect(300, 0, 1024, 588)
	screen.deckDropArea = dragdrop.NewDropArea(
		deckAreaBounds,
		[]string{"*"}, // Accept any card
		screen.handleCardDrop,
	)
	screen.dragManager.RegisterDroppable(screen.deckDropArea)

	// Convert collection buttons to draggable items
	screen.createDraggableItems(collectionButtons)

	// Load deck cards for display
	err = screen.loadDeckCards()
	if err != nil {
		return nil, fmt.Errorf("failed to load deck cards: %w", err)
	}

	return screen, nil
}

// createCollectionButtons creates buttons from the player's card collection
func (s *EditDeckScreen) createCollectionButtons() ([]*elements.Button, error) {
	buttons := make([]*elements.Button, 0)

	// Group cards by name
	for card, item := range s.Player.CardCollection {
		if item.Count > 0 {
			cardName := card.Name()
			if group, exists := s.collectionGroups[cardName]; exists {
				// Add to existing group
				group.cards = append(group.cards, card)
				group.totalCount += item.Count
			} else {
				// Create new group
				s.collectionGroups[cardName] = &cardGroup{
					name:       cardName,
					cards:      []*domain.Card{card},
					totalCount: item.Count,
				}
			}
		}
	}

	// Convert groups to sorted list
	groupList := make([]*cardGroup, 0, len(s.collectionGroups))
	for _, group := range s.collectionGroups {
		groupList = append(groupList, group)
	}

	// Sort groups by name
	sort.Slice(groupList, func(i, j int) bool {
		return groupList[i].name < groupList[j].name
	})

	// Create a button for each card group
	for _, group := range groupList {
		// Use first card in group for image
		representativeCard := group.cards[0]
		cardImg, err := representativeCard.CardImage(domain.CardViewArtOnly)
		if err != nil {
			fmt.Printf("WARN: Unable to load card image for %s: %v\n", group.name, err)
			continue
		}

		// Scale card image to fit in the collection list
		scaledCard := imageutil.ScaleImage(cardImg, 0.6)

		// Create button at position 0,0 (ScrollableList will position it)
		btn := elements.NewButton(scaledCard, scaledCard, scaledCard, 0, 0, 1.0)
		btn.ID = group.name

		buttons = append(buttons, btn)
	}

	return buttons, nil
}

// createDraggableItems wraps collection buttons as draggable items
func (s *EditDeckScreen) createDraggableItems(buttons []*elements.Button) {
	for _, btn := range buttons {
		// Get the card group for this button
		group, exists := s.collectionGroups[btn.ID]
		if exists && len(group.cards) > 0 {
			// Use first available card from the group as representative
			draggableBtn := dragdrop.NewDraggableButton(btn, group.cards[0])
			s.draggableItems = append(s.draggableItems, draggableBtn)
		}
	}
}

// Draw renders the edit deck screen
func (s *EditDeckScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	// Calculate position for collection list at bottom of screen
	collectionY := H - COLLECTION_HEIGHT

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate(0, float64(collectionY))

	// Draw the scrollable collection list
	s.CollectionList.Draw(screen, opts, scale)

	// Draw the deck drop area highlight if hovering
	s.deckDropArea.Draw(screen)

	// Draw deck cards
	s.drawDeckCards(screen, scale)

	// Draw the magnifier image if it exists
	if s.MagnifierImage != nil {
		magOpts := &ebiten.DrawImageOptions{}
		magOpts.GeoM.Scale(scale, scale)
		// Position on the left side, vertically centered in the space above the collection list
		// Available height is H - COLLECTION_HEIGHT
		// Image is 300x419
		magX := 1.0
		magY := (float64(H-COLLECTION_HEIGHT) - 419.0) / 2.0
		if magY < 10 {
			magY = 10
		}
		magOpts.GeoM.Translate(magX*scale, magY*scale)
		screen.DrawImage(s.MagnifierImage, magOpts)
	}

	// Draw deck area outline
	deckBounds := s.deckDropArea.GetDropBounds()
	deckOutline := ebiten.NewImage(deckBounds.Dx(), deckBounds.Dy())
	for y := 0; y < deckBounds.Dy(); y++ {
		for x := 0; x < deckBounds.Dx(); x++ {
			if x < 3 || x >= deckBounds.Dx()-3 || y < 3 || y >= deckBounds.Dy()-3 {
				deckOutline.Set(x, y, color.RGBA{100, 100, 100, 128})
			}
		}
	}
	deckOpts := &ebiten.DrawImageOptions{}
	deckOpts.GeoM.Scale(scale, scale)
	deckOpts.GeoM.Translate(float64(deckBounds.Min.X)*scale, float64(deckBounds.Min.Y)*scale)
	screen.DrawImage(deckOutline, deckOpts)

	// Draw drag image if dragging
	s.dragManager.Draw(screen)
}

// drawDeckCards draws the cards in the deck area with count overlays
func (s *EditDeckScreen) drawDeckCards(screen *ebiten.Image, scale float64) {
	for _, display := range s.deckCardDisplays {
		if display.Image == nil {
			continue
		}

		// Draw the card image
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Scale(scale, scale)
		cardOpts.GeoM.Translate(float64(display.X)*scale, float64(display.Y)*scale)
		screen.DrawImage(display.Image, cardOpts)

		// Draw count if more than 1
		if display.Count > 1 {
			countStr := strconv.Itoa(display.Count)

			// Create font face
			face := &text.GoTextFace{
				Source: fonts.MtgFont,
				Size:   24,
			}

			// Calculate position for count (lower right corner of card)
			cardWidth := display.Image.Bounds().Dx()
			cardHeight := display.Image.Bounds().Dy()

			// Create background for count
			countBgSize := 30
			countBg := ebiten.NewImage(countBgSize, countBgSize)

			// Draw semi-transparent black circle background
			for y := 0; y < countBgSize; y++ {
				for x := 0; x < countBgSize; x++ {
					dx := x - countBgSize/2
					dy := y - countBgSize/2
					if dx*dx+dy*dy <= (countBgSize/2)*(countBgSize/2) {
						countBg.Set(x, y, color.RGBA{0, 0, 0, 200})
					}
				}
			}

			// Draw count background
			bgOpts := &ebiten.DrawImageOptions{}
			bgOpts.GeoM.Scale(scale, scale)
			countX := float64(display.X + cardWidth - countBgSize - 5)
			countY := float64(display.Y + cardHeight - countBgSize - 5)
			bgOpts.GeoM.Translate(countX*scale, countY*scale)
			screen.DrawImage(countBg, bgOpts)

			// Draw count text
			textOpts := &ebiten.DrawImageOptions{}
			textOpts.GeoM.Scale(scale, scale)
			// Center text in the circle
			textX := countX + float64(countBgSize/2) - 8
			textY := countY + float64(countBgSize/2) + 8
			textOpts.GeoM.Translate(textX*scale, textY*scale)
			textOpts.ColorScale.Scale(1, 1, 1, 1) // White text
			fonts.DrawText(screen, countStr, face, textOpts)
		}
	}
}

// Update handles user interactions
func (s *EditDeckScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	s.clickFrame++

	// Calculate position for collection list
	collectionY := H - COLLECTION_HEIGHT

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(0, float64(collectionY))

	// Update the scrollable list
	s.CollectionList.Update(opts, scale, W, H)

	// Update drag and drop system
	mx, my := ebiten.CursorPosition()
	leftPressed := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	leftJustReleased := inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)

	// Convert draggable buttons to interface slice for drag manager
	draggables := make([]dragdrop.Draggable, len(s.draggableItems))
	for i, item := range s.draggableItems {
		draggables[i] = item
	}

	s.dragManager.Update(mx, my, leftPressed, leftJustReleased, draggables)

	// Check for hover and double-clicks on collection cards
	for i, btn := range s.CollectionList.GetItems() {
		// Handle hover for magnifier
		if btn.State == elements.StateHover {
			card := domain.FindCardByName(btn.ID)
			if card != nil {
				img, err := card.CardImage(domain.CardViewFull)
				if err == nil {
					s.MagnifierImage = img
				}
			}
		}

		if btn.IsClicked() {
			// Check if this is a double-click
			lastClick, exists := s.lastClickTime[i]
			if exists && (s.clickFrame-lastClick) < 30 { // 30 frames = ~0.5 seconds at 60fps
				// Double-click detected
				s.handleCardDoubleClick(i)
				delete(s.lastClickTime, i)
			} else {
				// Single click - record time
				s.lastClickTime[i] = s.clickFrame
			}
			btn.State = elements.StateNormal
		}
	}

	// Handle escape key to return to previous screen
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return screenui.CityScr, nil, nil
	}

	return screenui.EditDeckScr, nil, nil
}

// handleCardDoubleClick adds a card from collection to the deck
func (s *EditDeckScreen) handleCardDoubleClick(cardIdx int) {
	items := s.CollectionList.GetItems()
	if cardIdx >= len(items) {
		return
	}

	btn := items[cardIdx]
	// Get the card group for this button
	group, exists := s.collectionGroups[btn.ID]
	if !exists || len(group.cards) == 0 {
		fmt.Printf("No card group found for %s\n", btn.ID)
		return
	}

	// Find an available card from the group to add to deck
	var cardToAdd *domain.Card
	for _, card := range group.cards {
		collectionCount := s.Player.CardCollection.GetTotalCount(card)
		if collectionCount <= 0 {
			continue
		}

		// Check how many of this specific printing are already in deck
		deckCount := s.Player.CardCollection.GetDeckCount(card, s.Player.ActiveDeck)
		if deckCount < collectionCount {
			cardToAdd = card
			break
		}
	}

	if cardToAdd == nil {
		fmt.Printf("All copies of %s are already in deck\n", group.name)
		return
	}

	// Add card to deck
	s.Player.CardCollection.AddCardToDeck(cardToAdd, s.Player.ActiveDeck, 1)
	newCount := s.Player.CardCollection.GetDeckCount(cardToAdd, s.Player.ActiveDeck)
	fmt.Printf("Added %s to deck (now %d copies)\n", cardToAdd.Name(), newCount)

	// Reload deck display
	err := s.loadDeckCards()
	if err != nil {
		fmt.Printf("Error reloading deck cards: %v\n", err)
	}
}

// loadDeckCards loads the player's deck cards for display
func (s *EditDeckScreen) loadDeckCards() error {
	s.deckCardDisplays = make([]DeckCardDisplay, 0)

	// Group deck cards by name
	deckGroups := make(map[string]*cardGroup)
	for card, count := range s.Player.Character.GetDeck(s.Player.ActiveDeck) {
		if count > 0 {
			cardName := card.Name()
			if group, exists := deckGroups[cardName]; exists {
				// Add to existing group
				group.cards = append(group.cards, card)
				group.totalCount += count
			} else {
				// Create new group
				deckGroups[cardName] = &cardGroup{
					name:       cardName,
					cards:      []*domain.Card{card},
					totalCount: count,
				}
			}
		}
	}

	// Convert to sorted list
	groupList := make([]*cardGroup, 0, len(deckGroups))
	for _, group := range deckGroups {
		groupList = append(groupList, group)
	}

	// Sort groups by name for consistent ordering
	sort.Slice(groupList, func(i, j int) bool {
		return groupList[i].name < groupList[j].name
	})

	// Load and cache card images
	for _, group := range groupList {
		// Use first card in group for image
		representativeCard := group.cards[0]

		// Check if we already have this card image cached
		cachedImg, exists := s.deckCardImages[group.name]
		if !exists {
			// Load the card art image
			cardImg, err := representativeCard.CardImage(domain.CardViewArtOnly)
			if err != nil {
				fmt.Printf("WARN: Unable to load deck card image for %s: %v\n", group.name, err)
				continue
			}

			// Scale the card image for deck display (smaller than collection)
			scaledImg := imageutil.ScaleImage(cardImg, 0.4)
			s.deckCardImages[group.name] = scaledImg
			cachedImg = scaledImg
		}

		// Add to deck display list
		s.deckCardDisplays = append(s.deckCardDisplays, DeckCardDisplay{
			Card:  representativeCard,
			Count: group.totalCount,
			Image: cachedImg,
		})
	}

	// Calculate positions for deck cards
	s.calculateDeckCardPositions()

	return nil
}

// calculateDeckCardPositions calculates grid positions for deck cards
func (s *EditDeckScreen) calculateDeckCardPositions() {
	if len(s.deckCardDisplays) == 0 {
		return
	}

	bounds := s.deckDropArea.GetDropBounds()

	// Card dimensions (scaled)
	cardWidth := 100  // Approximate width after 0.4 scale
	cardHeight := 140 // Approximate height after 0.4 scale
	padding := 10

	// Calculate grid dimensions
	areaWidth := bounds.Dx() - 20 // Leave some margin

	cols := areaWidth / (cardWidth + padding)
	if cols == 0 {
		cols = 1
	}

	// Position cards in grid
	for i := range s.deckCardDisplays {
		col := i % cols
		row := i / cols

		x := bounds.Min.X + 10 + col*(cardWidth+padding)
		y := bounds.Min.Y + 10 + row*(cardHeight+padding)

		s.deckCardDisplays[i].X = x
		s.deckCardDisplays[i].Y = y
	}
}

// handleCardDrop handles when a card is dropped in the deck area
func (s *EditDeckScreen) handleCardDrop(data dragdrop.DragData) bool {
	cardData, ok := data.(*dragdrop.CardDragData)
	if !ok {
		return false
	}

	droppedCard, ok := cardData.Card.(*domain.Card)
	if !ok {
		return false
	}

	// Get the card group for this card
	group, exists := s.collectionGroups[droppedCard.Name()]
	if !exists || len(group.cards) == 0 {
		fmt.Printf("No card group found for %s\n", droppedCard.Name())
		return false
	}

	// Find an available card from the group to add to deck
	var cardToAdd *domain.Card
	for _, card := range group.cards {
		collectionCount := s.Player.CardCollection.GetTotalCount(card)
		if collectionCount <= 0 {
			continue
		}

		// Check how many of this specific printing are already in deck
		deckCount := s.Player.CardCollection.GetDeckCount(card, s.Player.ActiveDeck)
		if deckCount < collectionCount {
			cardToAdd = card
			break
		}
	}

	if cardToAdd == nil {
		fmt.Printf("All copies of %s are already in deck\n", group.name)
		return false
	}

	// Add card to deck
	s.Player.CardCollection.AddCardToDeck(cardToAdd, s.Player.ActiveDeck, 1)
	newCount := s.Player.CardCollection.GetDeckCount(cardToAdd, s.Player.ActiveDeck)
	fmt.Printf("Added %s to deck via drag (now %d copies)\n", cardToAdd.Name(), newCount)

	// Reload deck display
	err := s.loadDeckCards()
	if err != nil {
		fmt.Printf("Error reloading deck cards: %v\n", err)
	}

	return true
}
