package screens

import (
	"fmt"
	"image"
	"image/color"
	"sort"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	COLLECTION_WIDTH  = 1024
	COLLECTION_HEIGHT = 180
)

// EditDeckScreen allows players to edit their decks
type EditDeckScreen struct {
	Player         *domain.Player
	CollectionList *elements.ScrollableList
	Background     *ebiten.Image
	DeckButtons    []*elements.Button // Buttons for cards currently in the deck
	lastClickTime  map[int]int        // Track click times for double-click detection
	clickFrame     int                // Current frame counter for double-click timing
	MagnifierImage *ebiten.Image      // Image to display in the magnifier
	dragManager    *dragdrop.DragManager
	deckDropArea   *dragdrop.DropArea
	draggableItems []*dragdrop.DraggableButton
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
		Player:         player,
		Background:     collectionBg,
		DeckButtons:    make([]*elements.Button, 0),
		lastClickTime:  make(map[int]int),
		clickFrame:     0,
		dragManager:    dragdrop.NewDragManager(),
		draggableItems: make([]*dragdrop.DraggableButton, 0),
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

	// Create deck drop area (center-top area of screen)
	deckAreaBounds := image.Rect(W/4, 50, 3*W/4, H-COLLECTION_HEIGHT-50)
	screen.deckDropArea = dragdrop.NewDropArea(
		deckAreaBounds,
		[]string{"*"}, // Accept any card
		screen.handleCardDrop,
	)
	screen.dragManager.RegisterDroppable(screen.deckDropArea)

	// Convert collection buttons to draggable items
	screen.createDraggableItems(collectionButtons)

	return screen, nil
}

// createCollectionButtons creates buttons from the player's card collection
func (s *EditDeckScreen) createCollectionButtons() ([]*elements.Button, error) {
	buttons := make([]*elements.Button, 0)

	// Convert map to slice for consistent ordering
	type cardCount struct {
		card  *domain.Card
		count int
	}

	cardList := make([]cardCount, 0, len(s.Player.CardCollection))
	for card, count := range s.Player.CardCollection {
		if count > 0 {
			cardList = append(cardList, cardCount{card: card, count: count})
		}
	}

	// Sort the card list by name
	sort.Slice(cardList, func(i, j int) bool {
		return cardList[i].card.Name() < cardList[j].card.Name()
	})

	// Create a button for each card in the collection
	for _, cc := range cardList {
		cardImg, err := cc.card.CardImage(domain.CardViewArtOnly)
		if err != nil {
			fmt.Printf("WARN: Unable to load card image for %s: %v\n", cc.card.Name(), err)
			continue
		}

		// Scale card image to fit in the collection list
		scaledCard := imageutil.ScaleImage(cardImg, 0.6)

		// Create button at position 0,0 (ScrollableList will position it)
		btn := elements.NewButton(scaledCard, scaledCard, scaledCard, 0, 0, 1.0)
		btn.ID = cc.card.Name()

		buttons = append(buttons, btn)
	}

	return buttons, nil
}

// createDraggableItems wraps collection buttons as draggable items
func (s *EditDeckScreen) createDraggableItems(buttons []*elements.Button) {
	for _, btn := range buttons {
		card := domain.FindCardByName(btn.ID)
		if card != nil {
			draggableBtn := dragdrop.NewDraggableButton(btn, card)
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

	// Draw the magnifier image if it exists
	if s.MagnifierImage != nil {
		magOpts := &ebiten.DrawImageOptions{}
		magOpts.GeoM.Scale(scale, scale)
		// Position on the left side, vertically centered in the space above the collection list
		// Available height is H - COLLECTION_HEIGHT
		// Image is 300x419
		magX := 50.0
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
	fmt.Printf("Double-clicked card at index %d\n", cardIdx)
	// TODO: Implement adding card to deck
	// 1. Get the card from the collection
	// 2. Add it to the active deck
	// 3. Update deck buttons display
	// 4. Potentially update collection display if count changes
}

// handleCardDrop handles when a card is dropped in the deck area
func (s *EditDeckScreen) handleCardDrop(data dragdrop.DragData) bool {
	cardData, ok := data.(*dragdrop.CardDragData)
	if !ok {
		return false
	}

	card, ok := cardData.Card.(*domain.Card)
	if !ok {
		return false
	}

	fmt.Printf("Adding card to deck: %s\n", card.Name())
	// TODO: Implement actual deck addition logic
	// 1. Check if player has this card in collection
	// 2. Check deck size limits
	// 3. Add card to active deck
	// 4. Update UI

	return true
}
