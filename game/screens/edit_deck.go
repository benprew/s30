package screens

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strconv"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui"
	"github.com/benprew/s30/game/ui/dragdrop"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/layout"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	COLLECTION_WIDTH     = 1024
	COLLECTION_HEIGHT    = 180
	deckCardDragIDPrefix = "deck:"
	deckVisibleRows      = 4
)

type cardGroup struct {
	name       string
	cards      []*domain.Card // All printings of this card
	totalCount int            // Total count across all printings
}

// EditDeckScreen allows players to edit their decks
type EditDeckScreen struct {
	DisableSelling     bool
	ReturnScr          screenui.ScreenName
	ReturnScreen       screenui.Screen
	Player             *domain.Player
	City               *domain.City
	CollectionList     *elements.ScrollableList
	Background         *ebiten.Image
	TiledBackground    *ebiten.Image
	DeckBackground     *ebiten.Image
	DeckButtons        []*elements.Button // Buttons for cards currently in the deck
	MagnifierImage     *ebiten.Image      // Image to display in the magnifier
	MagnifiedCard      *domain.Card       // Currently magnified card
	dragManager        *dragdrop.DragManager
	deckDropArea       *dragdrop.DropArea
	collectionDropArea *dragdrop.DropArea
	sellDropArea       *dragdrop.DropArea
	draggableItems     []*dragdrop.DraggableButton
	deckDraggableItems []*dragdrop.DraggableButton
	deckCardDisplays   []DeckCardDisplay // Cards currently in the deck with counts
	deckScrollRow      int               // First visible row of cards in the deck area

	// Track buttons that already have card art.
	refreshedArt         map[string]bool
	collectionGroups     map[string]*cardGroup // Map card name to group
	hoveredCollectionIdx int                   // Index of hovered collection card (-1 if none)
	hoveredDeckIdx       int                   // Index of hovered deck card (-1 if none)
	filter               collectionFilter      // Active color/type filters for the collection
	filterButtons        []*filterButton       // Sprite-sheet toggle buttons for the filter

	// The editor checks the active deck against the minimum before closing and
	// warns the player when it falls short, instead of letting the duel pad it
	// with random basic lands without a word.
	leaveWarning     bool
	leaveWarningBody string
	leaveWarningBtn  *elements.Button
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

func (s *EditDeckScreen) IsOverlay() bool { return false }

// NewEditDeckScreen creates a new edit deck screen
func NewEditDeckScreen(player *domain.Player, city *domain.City, W, H int) (*EditDeckScreen, error) {
	// Load the collection background
	collectionBg, err := imageutil.LoadImage(assets.EditDeckBg_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load edit deck background: %w", err)
	}

	tileImg, err := imageutil.LoadImage(assets.EditDeckTile_png)
	if err != nil {
		return nil, fmt.Errorf("failed to load edit deck tile: %w", err)
	}

	// Create deck area with fixed bounds
	deckAreaBounds := image.Rect(300, 0, 1024, 540)

	deckBackground, err := loadEditDeckBackground(deckAreaBounds)
	if err != nil {
		return nil, fmt.Errorf("failed to load edit deck terrain: %w", err)
	}

	screen := &EditDeckScreen{
		Player:               player,
		City:                 city,
		Background:           collectionBg,
		TiledBackground:      imageutil.TileImage(tileImg, 1024, 768),
		DeckBackground:       deckBackground,
		DeckButtons:          make([]*elements.Button, 0),
		dragManager:          dragdrop.NewDragManager(),
		draggableItems:       make([]*dragdrop.DraggableButton, 0),
		deckDraggableItems:   make([]*dragdrop.DraggableButton, 0),
		deckCardDisplays:     make([]DeckCardDisplay, 0),
		collectionGroups:     make(map[string]*cardGroup),
		hoveredCollectionIdx: -1,
		hoveredDeckIdx:       -1,
		filter:               newCollectionFilter(),
	}

	filterButtons, err := createFilterButtons()
	if err != nil {
		return nil, fmt.Errorf("failed to create filter buttons: %w", err)
	}
	screen.filterButtons = filterButtons

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
	screen.sellDropArea = dragdrop.NewDropArea(
		editDeckSellBounds(),
		[]string{"*"},
		screen.handleCardDropToSell,
	)
	screen.dragManager.RegisterDroppable(screen.sellDropArea)

	screen.deckDropArea = dragdrop.NewDropArea(
		deckAreaBounds,
		[]string{"*"},
		screen.handleCardDrop,
	)
	screen.deckDropArea.SetCanAccept(func(data dragdrop.DragData) bool {
		return !strings.HasPrefix(data.GetID(), deckCardDragIDPrefix)
	})
	screen.dragManager.RegisterDroppable(screen.deckDropArea)

	// Create collection area drop zone
	collectionY := H - COLLECTION_HEIGHT
	collectionAreaBounds := image.Rect(0, collectionY, COLLECTION_WIDTH, H)
	screen.collectionDropArea = dragdrop.NewDropArea(
		collectionAreaBounds,
		[]string{"*"},
		screen.handleCardDropToCollection,
	)
	screen.collectionDropArea.SetCanAccept(func(data dragdrop.DragData) bool {
		return strings.HasPrefix(data.GetID(), deckCardDragIDPrefix)
	})
	screen.dragManager.RegisterDroppable(screen.collectionDropArea)

	// Convert collection buttons to draggable items
	screen.createDraggableItems(collectionButtons)

	// Load deck cards for display
	err = screen.loadDeckCards()
	if err != nil {
		return nil, fmt.Errorf("failed to load deck cards: %w", err)
	}

	return screen, nil
}

func loadEditDeckBackground(deckAreaBounds image.Rectangle) (*ebiten.Image, error) {
	terrain, err := imageutil.LoadImage(assets.EditDeckTerrain_png)
	if err != nil {
		return nil, err
	}

	return imageutil.TileImage(terrain, deckAreaBounds.Dx(), deckAreaBounds.Dy()), nil
}

// createCollectionButtons creates buttons from the player's card collection
func (s *EditDeckScreen) createCollectionButtons() ([]*elements.Button, error) {
	buttons := make([]*elements.Button, 0)
	s.collectionGroups = make(map[string]*cardGroup)

	// Group cards by name and calculate available count
	for card, item := range s.Player.CardCollection {
		if item.Count > 0 && s.filter.matches(card) {
			cardName := card.Name()
			deckCount := s.Player.CardCollection.GetDeckCount(card, s.Player.ActiveDeck)
			availableCount := item.Count - deckCount

			if availableCount > 0 {
				if group, exists := s.collectionGroups[cardName]; exists {
					group.cards = append(group.cards, card)
					group.totalCount += availableCount
				} else {
					s.collectionGroups[cardName] = &cardGroup{
						name:       cardName,
						cards:      []*domain.Card{card},
						totalCount: availableCount,
					}
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
		representativeCard := group.cards[0]
		cardImg, err := representativeCard.CardImage(domain.CardViewArtOnly)
		if err != nil {
			fmt.Printf("WARN: Unable to load card image for %s: %v\n", group.name, err)
			continue
		}

		btn := elements.NewButton(cardImg, cardImg, cardImg, 0, 0, 1.0)
		btn.ID = group.name

		buttons = append(buttons, btn)
	}

	return buttons, nil
}

// refreshCollectionImages replaces images when card art becomes available.
func (s *EditDeckScreen) refreshCollectionImages() {
	if s.refreshedArt == nil {
		s.refreshedArt = make(map[string]bool)
	}
	for _, btn := range s.CollectionList.GetItems() {
		if s.refreshedArt[btn.ID] {
			continue
		}
		group, ok := s.collectionGroups[btn.ID]
		if !ok || len(group.cards) == 0 {
			continue
		}
		card := group.cards[0]
		if !card.ImageLoaded() {
			continue
		}
		cardImg, err := card.CardImage(domain.CardViewArtOnly)
		if err != nil {
			continue
		}
		btn.Normal, btn.Hover, btn.Pressed = cardImg, cardImg, cardImg
		s.refreshedArt[btn.ID] = true
	}
}

func (s *EditDeckScreen) refreshDeckImages() {
	for i := range s.deckCardDisplays {
		display := &s.deckCardDisplays[i]
		img, err := display.Card.CardImage(domain.CardViewArtMini)
		if err != nil || img == display.Image {
			continue
		}
		display.Image = img
		for _, btn := range s.deckDraggableItems {
			if btn.ID == deckCardDragIDPrefix+display.Card.Name() {
				btn.Normal, btn.Hover, btn.Pressed = img, img, img
			}
		}
	}
}

// createDraggableItems wraps collection buttons as draggable items
func (s *EditDeckScreen) createDraggableItems(buttons []*elements.Button) {
	for _, btn := range buttons {
		group, exists := s.collectionGroups[btn.ID]
		if exists && len(group.cards) > 0 {
			draggableBtn := dragdrop.NewDraggableButton(btn, group.cards[0])
			s.draggableItems = append(s.draggableItems, draggableBtn)
		}
	}
}

// reloadCollectionList rebuilds the collection list after deck changes
func (s *EditDeckScreen) reloadCollectionList() error {
	s.refreshedArt = nil

	collectionButtons, err := s.createCollectionButtons()
	if err != nil {
		return fmt.Errorf("failed to create collection buttons: %w", err)
	}

	s.CollectionList.SetItems(collectionButtons)

	s.draggableItems = make([]*dragdrop.DraggableButton, 0)
	s.createDraggableItems(collectionButtons)

	return nil
}

// Draw renders the edit deck screen
func (s *EditDeckScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	baseOpts := ebiten.DrawImageOptions{}
	baseOpts.GeoM.Scale(scale, scale)

	dckOpts := baseOpts
	b := s.deckDropArea.GetDropBounds()
	dckOpts.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))

	screen.DrawImage(s.TiledBackground, &baseOpts)

	screen.DrawImage(s.DeckBackground, &dckOpts)

	// Calculate position for collection list at bottom of screen
	collectionY := H - COLLECTION_HEIGHT

	opts := baseOpts
	opts.GeoM.Translate(0, float64(collectionY))
	s.CollectionList.Draw(screen, &opts, scale)
	s.drawCollectionCounts(screen, scale, collectionY)

	if !s.DisableSelling {
		drawDeckSellTarget(screen, editDeckSellBounds(), s.Player.Gold)
	}

	s.drawDeckCards(screen, scale)
	s.drawDeckStats(screen, scale)
	s.drawDeckScrollButtons(screen, scale)
	s.drawFilterButtons(screen, scale)
	s.drawFilterCue(screen, W, H, scale)

	// Draw the magnifier image if it exists
	if s.MagnifierImage != nil {
		magOpts := &ebiten.DrawImageOptions{}
		magOpts.GeoM.Scale(scale, scale)
		magX := 1.0
		// The left panel stacks the sell target, the hint under it, the sale price
		// and then the card. The shift keeps the price line clear of the hint above
		// it and off the card's top edge.
		magY := (float64(H-COLLECTION_HEIGHT)-419.0)/2.0 + 12
		if magY < 10 {
			magY = 10
		}
		magOpts.GeoM.Translate(magX*scale, magY*scale)
		screen.DrawImage(s.MagnifierImage, magOpts)

		if s.MagnifiedCard != nil && !s.DisableSelling {
			salePrice := s.MagnifiedCard.SalePrice(s.City)
			priceText := fmt.Sprintf("Sale Price: %d gold", salePrice)
			textX := magX + 10
			textY := magY - 25
			if textY < 0 {
				textY = magY + 430
			}
			scaleOpts := &ebiten.DrawImageOptions{}
			scaleOpts.GeoM.Scale(scale, scale)
			elements.NewText(20, priceText, int(textX), int(textY)).Draw(screen, scaleOpts, scale)
		}
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

	// Draw helper text above collection area
	helpText := "[A] Add to Deck\n[D] Remove from Deck\n[S] Sell Card"
	if s.DisableSelling {
		helpText = fmt.Sprintf("[A] Add to Deck\n[D] Remove from Deck\nMin deck: %d cards", s.Player.MinDeckSize)
	}

	helpY := float64(collectionY) - 130
	helpOpts := &ebiten.DrawImageOptions{}
	elements.NewText(14, helpText, int(10*scale), int(helpY*scale)).Draw(screen, helpOpts, 1.0)
	drawDeckActionButton(screen, editDeckBackBounds(W), "Back")

	if s.leaveWarning {
		s.drawLeaveWarning(screen, scale)
	}

	// Draw drag image if dragging
	s.dragManager.Draw(screen)
}

// drawDeckCards draws the cards in the deck area with count overlays
func (s *EditDeckScreen) drawDeckCards(screen *ebiten.Image, scale float64) {
	for _, display := range s.deckCardDisplays {
		if display.Image == nil || !s.isDeckCardVisible(display) {
			continue
		}

		// Draw the card image
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Scale(scale, scale)
		cardOpts.GeoM.Translate(float64(display.X)*scale, float64(display.Y)*scale)
		screen.DrawImage(display.Image, cardOpts)

		// Draw count if more than 1
		if display.Count > 1 {
			cardWidth := display.Image.Bounds().Dx()
			cardHeight := display.Image.Bounds().Dy()
			s.drawCountOverlay(screen, scale, display.X, display.Y, cardWidth, cardHeight, display.Count)
		}
	}
}

// Update handles user interactions
func (s *EditDeckScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if s.leaveWarning {
		return s.updateLeaveWarning(W, H, scale)
	}

	// Calculate position for collection list
	collectionY := H - COLLECTION_HEIGHT

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(0, float64(collectionY))

	// Update the scrollable list
	s.CollectionList.Update(opts, scale, W, H)

	// Update collection filter toggles. A change rebuilds the list and resets scroll.
	filterOpts := &ebiten.DrawImageOptions{}
	if s.updateFilterButtons(filterOpts, scale, W, H) {
		if err := s.reloadCollectionList(); err != nil {
			fmt.Printf("Error reloading collection list after filter: %v\n", err)
		}
		s.CollectionList.ResetScroll()
	}

	s.refreshCollectionImages()
	s.refreshDeckImages()

	if drag, started := ui.DragStart(); started {
		s.dragManager.Start(drag, s.visibleDraggables())
	}
	if drag, dragging := ui.Dragging(); dragging {
		s.dragManager.Move(drag)
	}
	if drag, ended := ui.DragEnd(); ended {
		s.dragManager.End(drag)
	}

	// Check for hover on collection cards
	for i, btn := range s.CollectionList.GetItems() {
		if btn.State == elements.StateHover {
			s.hoveredCollectionIdx = i
			s.hoveredDeckIdx = -1
			card := domain.FindCardByName(btn.ID)
			if card == nil {
				fmt.Println("Error finding card for Button:", btn.ID)
			}
			if card != nil {
				img, err := card.CardImage(domain.CardViewFull)
				if err == nil {
					s.MagnifierImage = img
					s.MagnifiedCard = card
				}
			}
		}
	}

	// Handle deck area scrolling (mouse wheel, navigation buttons, PageUp/PageDown)
	pointerPosition := ui.Position()
	scaledMX := int(float64(pointerPosition.X) / scale)
	scaledMY := int(float64(pointerPosition.Y) / scale)
	pointerPt := image.Point{X: scaledMX, Y: scaledMY}

	deckBounds := s.deckDropArea.GetDropBounds()
	if pointerPt.In(deckBounds) {
		_, wheelY := ebiten.Wheel()
		if wheelY < 0 {
			s.scrollDeckDown()
		} else if wheelY > 0 {
			s.scrollDeckUp()
		}
	}

	if s.maxDeckScrollRow() > 0 {
		if ui.Click(deckScrollUpBounds()) {
			s.scrollDeckUp()
		} else if ui.Click(deckScrollDownBounds()) {
			s.scrollDeckDown()
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyPageDown) {
		s.scrollDeckDown()
	} else if inpututil.IsKeyJustPressed(ebiten.KeyPageUp) {
		s.scrollDeckUp()
	}

	// Check for hover on deck cards (only visible ones)
	for i, display := range s.deckCardDisplays {
		if display.Image == nil || !s.isDeckCardVisible(display) {
			continue
		}
		cardWidth := display.Image.Bounds().Dx()
		cardHeight := display.Image.Bounds().Dy()
		if scaledMX >= display.X && scaledMX < display.X+cardWidth &&
			scaledMY >= display.Y && scaledMY < display.Y+cardHeight {
			s.hoveredDeckIdx = i
			s.hoveredCollectionIdx = -1
			img, err := display.Card.CardImage(domain.CardViewFull)
			if err == nil {
				s.MagnifierImage = img
				s.MagnifiedCard = display.Card
			}
			break
		}
	}

	// Handle keyboard shortcuts
	if inpututil.IsKeyJustPressed(ebiten.KeyA) && s.hoveredCollectionIdx >= 0 {
		s.handleDeckCardAdd(s.hoveredCollectionIdx)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) && s.hoveredDeckIdx >= 0 {
		s.handleDeckCardRemove(s.hoveredDeckIdx)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		s.sellHoveredCard()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || ui.Click(editDeckBackBounds(W)) {
		name := s.leaveEditor()
		if name != screenui.EditDeckScr {
			return name, s.ReturnScreen, nil
		}
		return name, nil, nil
	}

	return screenui.EditDeckScr, nil, nil
}

func (s *EditDeckScreen) visibleDraggables() []dragdrop.Draggable {
	buttons := s.CollectionList.GetItems()
	start := s.CollectionList.GetCurrentOffset()
	end := min(start+s.CollectionList.GetVisibleCount(), len(buttons))
	visible := make(map[*elements.Button]bool, end-start)
	for _, button := range buttons[start:end] {
		visible[button] = true
	}

	items := make([]dragdrop.Draggable, 0, len(visible)+len(s.deckDraggableItems))
	for _, item := range s.draggableItems {
		if visible[item.Button] {
			items = append(items, item)
		}
	}
	bounds := defaultDeckAreaBounds()
	if s.deckDropArea != nil {
		bounds = s.deckDropArea.GetDropBounds()
	}
	cardHeight := domain.CardArtMiniHeight
	for _, item := range s.deckDraggableItems {
		if item.Button.Bounds.Empty() || (item.Button.Bounds.Min.Y >= bounds.Min.Y+55 && item.Button.Bounds.Min.Y+cardHeight <= bounds.Max.Y) {
			items = append(items, item)
		}
	}
	return items
}

func (s *EditDeckScreen) sellHoveredCard() {
	if s.hoveredCollectionIdx >= 0 {
		s.handleSellCard(s.hoveredCollectionIdx, true)
	} else if s.hoveredDeckIdx >= 0 {
		s.handleSellCard(s.hoveredDeckIdx, false)
	}
}

// editDeckBackBounds places the exit button in the filter row's band, at its right
// end, which is where the original keeps its Done. The screen's logical height is
// fixed at 768 by Game.Layout.
func editDeckBackBounds(W int) image.Rectangle {
	const height = 26
	top := 768 - COLLECTION_HEIGHT - filterBtnSize - filterGapAboveCarousel + (filterBtnSize-height)/2
	return image.Rect(W-110, top, W-12, top+height)
}

func editDeckSellBounds() image.Rectangle { return image.Rect(10, 12, 150, 68) }

// leaveEditor decides where the screen goes when the player asks to leave. A deck
// below the minimum does not block the exit: the player is warned first, and the
// duel fills the shortfall with random basic lands either way.
func (s *EditDeckScreen) leaveEditor() screenui.ScreenName {
	if shortfall := s.Player.DeckShortfall(); shortfall > 0 {
		s.leaveWarning = true
		s.leaveWarningBody = deckShortfallMessage(s.Player.MinDeckSize-shortfall, s.Player.MinDeckSize)
		s.leaveWarningBtn = newLeaveWarningButton()
		return screenui.EditDeckScr
	}

	return s.returnScreenName()
}

func (s *EditDeckScreen) returnScreenName() screenui.ScreenName {
	if s.ReturnScr != 0 {
		return s.ReturnScr
	}
	return screenui.CityScr
}

// deckShortfallMessage names both sizes so the player knows how far the deck is
// from the minimum and what becomes of the gap.
func deckShortfallMessage(deckSize, minDeckSize int) string {
	return fmt.Sprintf("Your deck needs at least %d cards; it has %d.\nMissing cards are filled with random basic lands when you duel.",
		minDeckSize, deckSize)
}

// updateLeaveWarning runs while the warning is up: the exit only completes after
// the player has seen it. Escape dismisses it the same way its button does.
func (s *EditDeckScreen) updateLeaveWarning(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	opts := &ebiten.DrawImageOptions{}
	s.leaveWarningBtn.Update(opts, scale, W, H)

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || s.leaveWarningBtn.IsClicked() {
		s.leaveWarning = false
		s.leaveWarningBtn = nil
		return s.returnScreenName(), s.ReturnScreen, nil
	}

	return screenui.EditDeckScr, nil, nil
}

// drawLeaveWarning mirrors the dungeon's overlay: a dimmed screen, a panel, and
// the one button that closes it.
func (s *EditDeckScreen) drawLeaveWarning(screen *ebiten.Image, scale float64) {
	dim := ebiten.NewImage(1024, 768)
	dim.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 180})
	screen.DrawImage(dim, &ebiten.DrawImageOptions{})

	panelW, panelH := 620, 260
	px := (1024 - panelW) / 2
	py := (768 - panelH) / 2
	panel := ebiten.NewImage(panelW, panelH)
	panel.Fill(color.RGBA{R: 30, G: 25, B: 50, A: 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(px), float64(py))
	screen.DrawImage(panel, op)

	title := elements.NewText(28, "Deck Too Small", px+30, py+30)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	body := elements.NewText(20, s.leaveWarningBody, px+30, py+90)
	body.Color = color.RGBA{R: 230, G: 230, B: 240, A: 255}
	body.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	s.leaveWarningBtn.Draw(screen, &ebiten.DrawImageOptions{}, scale)
}

func newLeaveWarningButton() *elements.Button {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	font := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}
	label := "Continue"
	w, _ := elements.TextButtonSize(label, font)
	return elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: label, Font: font, ID: "continue",
		X: 512 - w/2, Y: 460,
	})
}

func drawDeckActionButton(screen *ebiten.Image, bounds image.Rectangle, label string) {
	button := ebiten.NewImage(bounds.Dx(), bounds.Dy())
	button.Fill(color.RGBA{R: 55, G: 45, B: 35, A: 240})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(bounds.Min.X), float64(bounds.Min.Y))
	screen.DrawImage(button, opts)
	elements.NewText(18, label, bounds.Min.X+22, bounds.Min.Y+10).Draw(screen, &ebiten.DrawImageOptions{}, 1)
}

// Reload deck display and collection list
func (s *EditDeckScreen) updateState() {
	err := s.loadDeckCards()
	if err != nil {
		fmt.Printf("Error reloading deck cards: %v\n", err)
	}
	err = s.reloadCollectionList()
	if err != nil {
		fmt.Printf("Error reloading collection list: %v\n", err)
	}
}

// handleDeckCardAdd adds a card from collection to the deck
func (s *EditDeckScreen) handleDeckCardAdd(cardIdx int) {
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

	// Move card to deck
	err := s.Player.CardCollection.MoveCardToDeck(cardToAdd, s.Player.ActiveDeck, 1)
	if err != nil {
		fmt.Printf("Error moving card to deck: %v\n", err)
		return
	}
	newCount := s.Player.CardCollection.GetDeckCount(cardToAdd, s.Player.ActiveDeck)
	fmt.Printf("Added %s to deck (now %d copies)\n", cardToAdd.Name(), newCount)
	s.updateState()
}

// handleDeckCardRemove removes a card from deck back to collection
func (s *EditDeckScreen) handleDeckCardRemove(deckIdx int) {
	if deckIdx >= len(s.deckCardDisplays) {
		return
	}

	display := s.deckCardDisplays[deckIdx]
	cardToRemove := display.Card

	err := s.Player.CardCollection.MoveCardFromDeck(cardToRemove, s.Player.ActiveDeck, 1)
	if err != nil {
		fmt.Printf("Error moving card from deck: %v\n", err)
		return
	}
	newCount := s.Player.CardCollection.GetDeckCount(cardToRemove, s.Player.ActiveDeck)
	fmt.Printf("Removed %s from deck (now %d copies)\n", cardToRemove.Name(), newCount)
	s.updateState()
}

func (s *EditDeckScreen) handleSellCard(cardIdx int, isFromCollection bool) {
	var cardToSell *domain.Card

	if isFromCollection {
		items := s.CollectionList.GetItems()
		if cardIdx >= len(items) {
			return
		}
		btn := items[cardIdx]
		group, exists := s.collectionGroups[btn.ID]
		if !exists || len(group.cards) == 0 {
			fmt.Printf("No card group found for %s\n", btn.ID)
			return
		}
		cardToSell = group.cards[0]
	} else {
		if cardIdx >= len(s.deckCardDisplays) {
			return
		}
		display := s.deckCardDisplays[cardIdx]
		cardToSell = display.Card
	}

	if s.sellCard(cardToSell, !isFromCollection) {
		s.updateState()
	}
}

func (s *EditDeckScreen) handleCardDropToSell(data dragdrop.DragData) bool {
	if !s.sellDroppedCard(data) {
		return false
	}
	s.updateState()
	return true
}

func (s *EditDeckScreen) sellDroppedCard(data dragdrop.DragData) bool {
	cardData, ok := data.(*dragdrop.CardDragData)
	if !ok {
		return false
	}
	card, ok := cardData.Card.(*domain.Card)
	if !ok {
		return false
	}
	return s.sellCard(card, strings.HasPrefix(cardData.ID, deckCardDragIDPrefix))
}

func (s *EditDeckScreen) sellCard(card *domain.Card, fromDeck bool) bool {
	if s.DisableSelling {
		return false
	}
	deckCount := s.Player.CardCollection.GetDeckCount(card, s.Player.ActiveDeck)
	if fromDeck {
		if deckCount == 0 {
			return false
		}
		if err := s.Player.CardCollection.MoveCardFromDeck(card, s.Player.ActiveDeck, 1); err != nil {
			fmt.Printf("Error moving card from deck before selling: %v\n", err)
			return false
		}
	} else if s.Player.CardCollection.GetTotalCount(card) <= deckCount {
		return false
	}

	salePrice := card.SalePrice(s.City)
	if err := s.Player.CardCollection.DecrementCardCount(card); err != nil {
		fmt.Printf("Error selling card: %v\n", err)
		return false
	}
	s.Player.Gold += salePrice
	fmt.Printf("Sold %s for %d gold\n", card.Name(), salePrice)
	return true
}

// loadDeckCards loads the player's deck cards for display
func (s *EditDeckScreen) loadDeckCards() error {
	s.deckCardDisplays = make([]DeckCardDisplay, 0)

	// Group deck cards by name
	deckGroups := make(map[string]*cardGroup)
	for card, count := range s.Player.GetDeck(s.Player.ActiveDeck) {
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

		cachedImg, err := representativeCard.CardImage(domain.CardViewArtMini)
		if err != nil {
			return err
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

	// Create draggable items for deck cards
	s.createDeckDraggableItems()

	return nil
}

func defaultDeckAreaBounds() image.Rectangle {
	return image.Rect(300, 0, 1024, 540)
}

// maxDeckScrollRow returns the maximum scroll row index
func (s *EditDeckScreen) maxDeckScrollRow() int {
	if len(s.deckCardDisplays) == 0 {
		return 0
	}
	bounds := defaultDeckAreaBounds()
	if s.deckDropArea != nil {
		bounds = s.deckDropArea.GetDropBounds()
	}
	cardWidth := domain.CardArtMiniWidth
	horizontalGap := 8
	areaWidth := bounds.Dx() - 20
	cols := (areaWidth + horizontalGap) / (cardWidth + horizontalGap)
	if cols == 0 {
		cols = 1
	}
	totalRows := (len(s.deckCardDisplays) + cols - 1) / cols
	return max(0, totalRows-deckVisibleRows)
}

func (s *EditDeckScreen) isDeckCardVisible(display DeckCardDisplay) bool {
	bounds := defaultDeckAreaBounds()
	if s.deckDropArea != nil {
		bounds = s.deckDropArea.GetDropBounds()
	}
	cardHeight := domain.CardArtMiniHeight
	return display.Y >= bounds.Min.Y+55 && display.Y+cardHeight <= bounds.Max.Y
}

func (s *EditDeckScreen) scrollDeckDown() {
	if s.deckScrollRow < s.maxDeckScrollRow() {
		s.deckScrollRow++
		s.calculateDeckCardPositions()
		s.updateDeckDraggablePositions()
	}
}

func (s *EditDeckScreen) scrollDeckUp() {
	if s.deckScrollRow > 0 {
		s.deckScrollRow--
		s.calculateDeckCardPositions()
		s.updateDeckDraggablePositions()
	}
}

func (s *EditDeckScreen) updateDeckDraggablePositions() {
	for i, display := range s.deckCardDisplays {
		if i < len(s.deckDraggableItems) {
			btn := s.deckDraggableItems[i].Button
			cardWidth := domain.CardArtMiniWidth
			cardHeight := domain.CardArtMiniHeight
			if display.Image != nil {
				cardWidth = display.Image.Bounds().Dx()
				cardHeight = display.Image.Bounds().Dy()
			}
			btn.Bounds = image.Rect(display.X, display.Y, display.X+cardWidth, display.Y+cardHeight)
		}
	}
}

func deckScrollUpBounds() image.Rectangle {
	return image.Rect(590, 485, 626, 515)
}

func deckScrollDownBounds() image.Rectangle {
	return image.Rect(710, 485, 746, 515)
}

func (s *EditDeckScreen) drawDeckScrollButtons(screen *ebiten.Image, scale float64) {
	if s.maxDeckScrollRow() == 0 {
		return
	}

	upBounds := deckScrollUpBounds()
	downBounds := deckScrollDownBounds()

	mPos := ui.Position()
	scaledMX := int(float64(mPos.X) / scale)
	scaledMY := int(float64(mPos.Y) / scale)
	pointerPt := image.Point{X: scaledMX, Y: scaledMY}

	upHover := pointerPt.In(upBounds)
	downHover := pointerPt.In(downBounds)

	canScrollUp := s.deckScrollRow > 0
	canScrollDown := s.deckScrollRow < s.maxDeckScrollRow()

	drawScrollButton(screen, upBounds, true, canScrollUp, upHover, scale)

	rowText := fmt.Sprintf("Row %d of %d", s.deckScrollRow+1, s.maxDeckScrollRow()+1)
	scaleOpts := &ebiten.DrawImageOptions{}
	scaleOpts.GeoM.Scale(scale, scale)
	txt := elements.NewText(14, rowText, 632, 492)
	txt.Color = color.RGBA{R: 200, G: 200, B: 210, A: 240}
	txt.Draw(screen, scaleOpts, scale)

	drawScrollButton(screen, downBounds, false, canScrollDown, downHover, scale)
}

func drawScrollButton(screen *ebiten.Image, bounds image.Rectangle, isUp bool, enabled bool, hover bool, scale float64) {
	bgColor := color.RGBA{R: 50, G: 45, B: 60, A: 240}
	if hover && enabled {
		bgColor = color.RGBA{R: 80, G: 70, B: 95, A: 240}
	} else if !enabled {
		bgColor = color.RGBA{R: 35, G: 30, B: 40, A: 180}
	}
	x := float32(bounds.Min.X) * float32(scale)
	y := float32(bounds.Min.Y) * float32(scale)
	w := float32(bounds.Dx()) * float32(scale)
	h := float32(bounds.Dy()) * float32(scale)
	vector.FillRect(screen, x, y, w, h, bgColor, false)

	borderColor := color.RGBA{R: 120, G: 110, B: 130, A: 200}
	if !enabled {
		borderColor = color.RGBA{R: 70, G: 65, B: 80, A: 150}
	}
	vector.StrokeRect(screen, x, y, w, h, 1, borderColor, false)

	arrowColor := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	if !enabled {
		arrowColor = color.RGBA{R: 100, G: 100, B: 110, A: 180}
	}

	centerX := float32(bounds.Min.X+bounds.Dx()/2) * float32(scale)
	centerY := float32(bounds.Min.Y+bounds.Dy()/2) * float32(scale)
	arm := float32(5) * float32(scale)

	if isUp {
		vector.StrokeLine(screen, centerX-arm, centerY+arm*0.5, centerX, centerY-arm*0.5, 2, arrowColor, false)
		vector.StrokeLine(screen, centerX, centerY-arm*0.5, centerX+arm, centerY+arm*0.5, 2, arrowColor, false)
	} else {
		vector.StrokeLine(screen, centerX-arm, centerY-arm*0.5, centerX, centerY+arm*0.5, 2, arrowColor, false)
		vector.StrokeLine(screen, centerX, centerY+arm*0.5, centerX+arm, centerY-arm*0.5, 2, arrowColor, false)
	}
}

// calculateDeckCardPositions calculates grid positions for deck cards
func (s *EditDeckScreen) calculateDeckCardPositions() {
	if len(s.deckCardDisplays) == 0 {
		return
	}

	bounds := defaultDeckAreaBounds()
	if s.deckDropArea != nil {
		bounds = s.deckDropArea.GetDropBounds()
	}

	cardWidth := domain.CardArtMiniWidth
	cardHeight := domain.CardArtMiniHeight
	padding := 10
	horizontalGap := 8

	// Calculate grid dimensions
	areaWidth := bounds.Dx() - 20 // Leave some margin

	cols := (areaWidth + horizontalGap) / (cardWidth + horizontalGap)
	if cols == 0 {
		cols = 1
	}

	maxScroll := s.maxDeckScrollRow()
	if s.deckScrollRow > maxScroll {
		s.deckScrollRow = maxScroll
	}
	if s.deckScrollRow < 0 {
		s.deckScrollRow = 0
	}

	// Position cards in grid
	for i := range s.deckCardDisplays {
		col := i % cols
		row := i / cols
		relRow := row - s.deckScrollRow

		x := bounds.Min.X + 10 + col*(cardWidth+horizontalGap)
		y := bounds.Min.Y + 55 + relRow*(cardHeight+padding)

		s.deckCardDisplays[i].X = x
		s.deckCardDisplays[i].Y = y
	}
}

// createDeckDraggableItems creates draggable buttons for deck cards
func (s *EditDeckScreen) createDeckDraggableItems() {
	s.deckDraggableItems = make([]*dragdrop.DraggableButton, 0)

	for _, display := range s.deckCardDisplays {
		if display.Image == nil {
			continue
		}

		btn := elements.NewButton(display.Image, display.Image, display.Image, display.X, display.Y, 1.0)
		btn.ID = deckCardDragIDPrefix + display.Card.Name()

		draggableBtn := dragdrop.NewDraggableButton(btn, display.Card)
		s.deckDraggableItems = append(s.deckDraggableItems, draggableBtn)
	}
}

// handleCardDrop handles when a card is dropped in the deck area
func (s *EditDeckScreen) handleCardDrop(data dragdrop.DragData) bool {
	if !s.addDroppedCardToDeck(data) {
		return false
	}
	s.updateState()
	return true
}

func (s *EditDeckScreen) addDroppedCardToDeck(data dragdrop.DragData) bool {
	cardData, ok := data.(*dragdrop.CardDragData)
	if !ok {
		return false
	}

	if strings.HasPrefix(cardData.ID, deckCardDragIDPrefix) {
		return false
	}

	droppedCard, ok := cardData.Card.(*domain.Card)
	if !ok {
		return false
	}

	group, exists := s.collectionGroups[droppedCard.Name()]
	if !exists || len(group.cards) == 0 {
		fmt.Printf("No card group found for %s\n", droppedCard.Name())
		return false
	}

	var cardToAdd *domain.Card
	for _, card := range group.cards {
		collectionCount := s.Player.CardCollection.GetTotalCount(card)
		if collectionCount <= 0 {
			continue
		}

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

	err := s.Player.CardCollection.MoveCardToDeck(cardToAdd, s.Player.ActiveDeck, 1)
	if err != nil {
		fmt.Printf("Error moving card to deck: %v\n", err)
		return false
	}
	newCount := s.Player.CardCollection.GetDeckCount(cardToAdd, s.Player.ActiveDeck)
	fmt.Printf("Added %s to deck via drag (now %d copies)\n", cardToAdd.Name(), newCount)

	return true
}

// handleCardDropToCollection handles when a card is dropped in the collection area
func (s *EditDeckScreen) handleCardDropToCollection(data dragdrop.DragData) bool {
	if !s.removeDroppedCardFromDeck(data) {
		return false
	}
	s.updateState()
	return true
}

func (s *EditDeckScreen) removeDroppedCardFromDeck(data dragdrop.DragData) bool {
	cardData, ok := data.(*dragdrop.CardDragData)
	if !ok {
		return false
	}

	if !strings.HasPrefix(cardData.ID, deckCardDragIDPrefix) {
		return false
	}

	droppedCard, ok := cardData.Card.(*domain.Card)
	if !ok {
		return false
	}

	err := s.Player.CardCollection.MoveCardFromDeck(droppedCard, s.Player.ActiveDeck, 1)
	if err != nil {
		fmt.Printf("Error moving card from deck: %v\n", err)
		return false
	}
	newCount := s.Player.CardCollection.GetDeckCount(droppedCard, s.Player.ActiveDeck)
	fmt.Printf("Removed %s from deck via drag (now %d copies)\n", droppedCard.Name(), newCount)

	return true
}

// drawCollectionCounts draws count overlays on collection cards
func (s *EditDeckScreen) drawCollectionCounts(screen *ebiten.Image, scale float64, collectionY int) {
	visibleItems := s.getVisibleCollectionItems(collectionY)

	for _, item := range visibleItems {
		group, exists := s.collectionGroups[item.ButtonID]
		if !exists || group.totalCount <= 1 {
			continue
		}

		// Draw count overlay using same logic as deck cards
		s.drawCountOverlay(screen, scale, item.X, item.Y, item.Width, item.Height, group.totalCount)
	}
}

type CollectionItemDisplay struct {
	ButtonID            string
	X, Y, Width, Height int
}

// getVisibleCollectionItems calculates positions of visible collection buttons
func (s *EditDeckScreen) getVisibleCollectionItems(collectionY int) []CollectionItemDisplay {
	items := s.CollectionList.GetItems()
	if len(items) == 0 {
		return nil
	}

	visibleItems := make([]CollectionItemDisplay, 0)

	// Get scroll state from collection list
	currentOffset := s.CollectionList.GetCurrentOffset()
	visibleCount := s.CollectionList.GetVisibleCount()

	// Calculate end index for visible items
	endIdx := min(currentOffset+visibleCount, len(items))

	// The items are positioned horizontally starting at the carousel's left
	// edge margin (must match horizontalEdgeMargin in ScrollableList)
	currentX := 10

	// Only process visible items
	for i := currentOffset; i < endIdx; i++ {
		item := items[i]
		buttonWidth := item.Bounds.Dx()
		buttonHeight := item.Bounds.Dy()

		visibleItems = append(visibleItems, CollectionItemDisplay{
			ButtonID: item.ID,
			X:        currentX,
			Y:        collectionY + 10, // match horizontalTopMargin in ScrollableList
			Width:    buttonWidth,
			Height:   buttonHeight,
		})

		currentX += buttonWidth + 15 // 15 is itemPadding
	}

	return visibleItems
}

// parseCMC parses a mana cost string like "{3}{G}{R}" and returns the total converted mana cost.
func parseCMC(manaCost string) int {
	cmc := 0
	i := 0
	for i < len(manaCost) {
		if manaCost[i] == '{' {
			j := i + 1
			for j < len(manaCost) && manaCost[j] != '}' {
				j++
			}
			if j < len(manaCost) {
				content := manaCost[i+1 : j]
				if intValue, err := strconv.Atoi(content); err == nil {
					cmc += intValue
				} else {
					cmc++
				}
				i = j + 1
			} else {
				i++
			}
		} else {
			i++
		}
	}
	return cmc
}

// drawDeckStats draws a stats header bar at the top of the deck area
func (s *EditDeckScreen) drawDeckStats(screen *ebiten.Image, scale float64) {
	if len(s.deckCardDisplays) == 0 {
		return
	}

	bounds := s.deckDropArea.GetDropBounds()

	totalCards := 0
	cmcCounts := make([]int, 8) // CMC 0 through 7+
	colorCounts := map[string]int{"W": 0, "U": 0, "B": 0, "R": 0, "G": 0}
	landCount := 0
	creatureCount := 0
	spellCount := 0

	for _, display := range s.deckCardDisplays {
		totalCards += display.Count

		if display.Card.ManaCost != "" {
			cmc := min(parseCMC(display.Card.ManaCost), 7)
			cmcCounts[cmc] += display.Count
		}

		for _, c := range display.Card.Colors {
			colorCounts[c] += display.Count
		}

		switch display.Card.CardType {
		case domain.CardTypeLand:
			landCount += display.Count
		case domain.CardTypeCreature:
			creatureCount += display.Count
		default:
			spellCount += display.Count
		}
	}

	scaleOpts := &ebiten.DrawImageOptions{}
	scaleOpts.GeoM.Scale(scale, scale)

	// Section 1: Card count
	countX := float64(bounds.Min.X + 10)
	elements.NewText(14, fmt.Sprintf("%d cards", totalCards), int(countX), 16).Draw(screen, scaleOpts, scale)

	// Section 2: Mana curve histogram
	curveX := bounds.Min.X + 80
	maxCount := 0
	for _, c := range cmcCounts {
		if c > maxCount {
			maxCount = c
		}
	}

	barWidth := 16
	barGap := 4
	maxBarHeight := 28.0

	for i, count := range cmcCounts {
		bx := curveX + i*(barWidth+barGap)

		if count > 0 && maxCount > 0 {
			barHeight := max(int(maxBarHeight*float64(count)/float64(maxCount)), 2)
			bar := ebiten.NewImage(barWidth, barHeight)
			bar.Fill(color.RGBA{180, 180, 220, 255})

			barOpts := &ebiten.DrawImageOptions{}
			barOpts.GeoM.Scale(scale, scale)
			barY := 42.0 - float64(barHeight)
			barOpts.GeoM.Translate(float64(bx)*scale, barY*scale)
			screen.DrawImage(bar, barOpts)
		}

		elements.NewText(10, strconv.Itoa(count), bx+2, 1).Draw(screen, scaleOpts, scale)

		cmcLabel := strconv.Itoa(i)
		if i == 7 {
			cmcLabel = "7+"
		}
		elements.NewText(10, cmcLabel, bx+2, 43).Draw(screen, scaleOpts, scale)
	}

	// Section 3: Color distribution
	colorX := curveX + 8*(barWidth+barGap) + 15
	colorRGBA := map[string]color.RGBA{
		"W": {248, 231, 185, 255},
		"U": {14, 104, 171, 255},
		"B": {166, 159, 157, 255},
		"R": {211, 32, 42, 255},
		"G": {0, 115, 62, 255},
	}
	colorOrder := []string{"W", "U", "B", "R", "G"}
	pipSize := 12

	for ci, c := range colorOrder {
		px := colorX + ci*36

		pip := ebiten.NewImage(pipSize, pipSize)
		pip.Fill(colorRGBA[c])
		pipOpts := &ebiten.DrawImageOptions{}
		pipOpts.GeoM.Scale(scale, scale)
		pipOpts.GeoM.Translate(float64(px)*scale, 18*scale)
		screen.DrawImage(pip, pipOpts)

		elements.NewText(14, strconv.Itoa(colorCounts[c]), px+pipSize+2, 16).Draw(screen, scaleOpts, scale)
	}

	// Section 4: Type counts
	typeX := float64(colorX + 5*36 + 15)
	typeText := fmt.Sprintf("Land: %d Creatures: %d Spells: %d", landCount, creatureCount, spellCount)
	elements.NewText(14, typeText, int(typeX), 16).Draw(screen, scaleOpts, scale)
}

// drawCountOverlay draws a count overlay at the specified position
func (s *EditDeckScreen) drawCountOverlay(screen *ebiten.Image, scale float64, x, y, width, height, count int) {
	countStr := strconv.Itoa(count)

	countBgSize := 30
	countBg := ebiten.NewImage(countBgSize, countBgSize)

	for bgY := range countBgSize {
		for bgX := range countBgSize {
			dx := bgX - countBgSize/2
			dy := bgY - countBgSize/2
			if dx*dx+dy*dy <= (countBgSize/2)*(countBgSize/2) {
				countBg.Set(bgX, bgY, color.RGBA{0, 0, 0, 200})
			}
		}
	}

	bgOpts := &ebiten.DrawImageOptions{}
	bgOpts.GeoM.Scale(scale, scale)
	countX := float64(x + width - countBgSize - 5)
	countY := float64(y + height - countBgSize - 5)
	bgOpts.GeoM.Translate(countX*scale, countY*scale)
	screen.DrawImage(countBg, bgOpts)

	scaleOpts := &ebiten.DrawImageOptions{}
	scaleOpts.GeoM.Scale(scale, scale)
	textX := countX + float64(countBgSize/2) - 7
	textY := countY + float64(countBgSize/2) - 12
	elements.NewText(24, countStr, int(textX), int(textY)).Draw(screen, scaleOpts, scale)
}
