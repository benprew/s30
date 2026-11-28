package screens

import (
	"cmp"
	"fmt"
	"image"
	"image/color"
	"maps"
	"slices"
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
	deckTopInset         = 55
	deckSideInset        = 10
	deckColumnGap        = 8
	deckRowGap           = 10
	deckMinRowStep       = 20
)

type cardGroup struct {
	owned      *domain.OwnedCard
	totalCount int
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
	deckPage           int
	deckPageSize       int

	// Track buttons that already have card art.
	refreshedArt     map[string]bool
	collectionGroups map[string]*cardGroup // Map card name to group
	filter           collectionFilter      // Active color/type filters for the collection
	filterButtons    []*filterButton       // Sprite-sheet toggle buttons for the filter

	// The editor checks the active deck against the minimum before closing and
	// warns the player when it falls short, instead of letting the duel pad it
	// with random basic lands without a word.
	leaveWarning     bool
	leaveWarningBody string
	leaveWarningBtn  *elements.Button
}

type DeckCardDisplay struct {
	Owned         *domain.OwnedCard
	Image         *ebiten.Image
	Count         int
	X             int
	Y             int
	colorRank     int
	manaValue     int
	visibleHeight int
}

func (d *DeckCardDisplay) visibleBounds() image.Rectangle {
	return image.Rect(d.X, d.Y, d.X+domain.CardArtMiniWidth, d.Y+d.visibleHeight)
}

func (d *DeckCardDisplay) cardImage() *ebiten.Image {
	if d.Image == nil && d.Owned != nil {
		d.Image, _ = d.Owned.CardImage(domain.CardViewArtMini)
	}
	return d.Image
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
		Player:             player,
		City:               city,
		Background:         collectionBg,
		TiledBackground:    imageutil.TileImage(tileImg, 1024, 768),
		DeckBackground:     deckBackground,
		DeckButtons:        make([]*elements.Button, 0),
		dragManager:        dragdrop.NewDragManager(),
		draggableItems:     make([]*dragdrop.DraggableButton, 0),
		deckDraggableItems: make([]*dragdrop.DraggableButton, 0),
		deckCardDisplays:   make([]DeckCardDisplay, 0),
		collectionGroups:   make(map[string]*cardGroup),
		filter:             newCollectionFilter(),
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
	screen.loadDeckCards()

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

	for card := range s.Player.CardCollection {
		if !s.filter.matches(card) {
			continue
		}
		available := s.Player.CardCollection.AvailableCards(card)
		if len(available) == 0 {
			continue
		}
		if group := s.collectionGroups[card.Name()]; group != nil {
			group.totalCount += len(available)
		} else {
			s.collectionGroups[card.Name()] = &cardGroup{owned: available[0], totalCount: len(available)}
		}
	}

	// Create a button for each card group
	for _, name := range slices.Sorted(maps.Keys(s.collectionGroups)) {
		group := s.collectionGroups[name]
		cardImg, err := group.owned.CardImage(domain.CardViewArtOnly)
		if err != nil {
			fmt.Printf("WARN: Unable to load card image for %s: %v\n", group.owned.Name(), err)
			continue
		}

		btn := elements.NewButton(cardImg, cardImg, cardImg, 0, 0, 1.0)
		btn.ID = name

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
		if !ok || group.owned == nil {
			continue
		}
		if !group.owned.ImageLoaded() {
			continue
		}
		cardImg, err := group.owned.CardImage(domain.CardViewArtOnly)
		if err != nil {
			continue
		}
		btn.Normal, btn.Hover, btn.Pressed = cardImg, cardImg, cardImg
		s.refreshedArt[btn.ID] = true
	}
}

func (s *EditDeckScreen) refreshDeckDragImages() {
	start, end := s.deckPageRange()
	for i := start; i < end; i++ {
		display := &s.deckCardDisplays[i]
		if !display.Owned.ImageLoaded() {
			continue
		}
		img, err := display.Owned.CardImage(domain.CardViewArtMini)
		if err != nil || img == nil {
			continue
		}
		display.Image = img
		id := deckCardDragIDPrefix + display.Owned.Name()
		for _, btn := range s.deckDraggableItems {
			if btn.ID == id {
				btn.Normal, btn.Hover, btn.Pressed = img, img, img
				break
			}
		}
	}
}

// createDraggableItems wraps collection buttons as draggable items
func (s *EditDeckScreen) createDraggableItems(buttons []*elements.Button) {
	for _, btn := range buttons {
		group, exists := s.collectionGroups[btn.ID]
		if exists && group.owned != nil {
			draggableBtn := dragdrop.NewDraggableButton(btn, group.owned.Card)
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
	s.drawDeckPaging(screen, W)
	drawDeckActionButton(screen, editDeckBackBounds(W), "Back")

	if s.leaveWarning {
		s.drawLeaveWarning(screen, scale)
	}

	// Draw drag image if dragging
	s.dragManager.Draw(screen)
}

// drawDeckCards draws the cards in the deck area with count overlays
func (s *EditDeckScreen) drawDeckCards(screen *ebiten.Image, scale float64) {
	const fullCountMinHeight = 48
	start, end := s.deckPageRange()
	for i := start; i < end; i++ {
		display := &s.deckCardDisplays[i]
		img := display.cardImage()
		if img == nil {
			continue
		}

		bounds := display.visibleBounds()
		imgBounds := img.Bounds()
		if bounds.Dy() < imgBounds.Dy() {
			imgBounds.Max.Y = imgBounds.Min.Y + bounds.Dy()
			img = img.SubImage(imgBounds).(*ebiten.Image)
		}
		cardOpts := &ebiten.DrawImageOptions{}
		cardOpts.GeoM.Scale(scale, scale)
		cardOpts.GeoM.Translate(float64(bounds.Min.X)*scale, float64(bounds.Min.Y)*scale)
		screen.DrawImage(img, cardOpts)

		if display.Count > 1 {
			if bounds.Dy() < fullCountMinHeight {
				drawCompactDeckCount(screen, scale, bounds, display.Count)
			} else {
				s.drawCountOverlay(screen, scale, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy(), display.Count)
			}
		}
	}
}

func drawCompactDeckCount(screen *ebiten.Image, scale float64, bounds image.Rectangle, count int) {
	const height = 10
	label := strconv.Itoa(count)
	face := &text.GoTextFace{Source: fonts.MtgFont, Size: 11}
	width, _ := text.Measure(label, face, 0)
	badgeWidth := int(width) + 5
	x := float64(bounds.Max.X - badgeWidth - 2)
	y := float64(bounds.Max.Y - height - 1)
	vector.FillRect(screen, float32(x*scale), float32(y*scale), float32(float64(badgeWidth)*scale),
		float32(height*scale), color.RGBA{A: 220}, false)
	opts := &text.DrawOptions{}
	opts.GeoM.Scale(scale, scale)
	opts.GeoM.Translate((x+2)*scale, (y-1)*scale)
	text.Draw(screen, label, face, opts)
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

	// Update collection filter toggles. A change rebuilds the list and resets scroll.
	filterOpts := &ebiten.DrawImageOptions{}
	if s.updateFilterButtons(filterOpts, scale, W, H) {
		if err := s.reloadCollectionList(); err != nil {
			fmt.Printf("Error reloading collection list after filter: %v\n", err)
		}
		s.CollectionList.ResetScroll()
	}

	s.CollectionList.Update(opts, scale, W, H)
	s.refreshCollectionImages()
	s.refreshDeckDragImages()

	if drag, started := ui.DragStart(); started {
		s.dragManager.Start(drag, s.visibleDraggables())
	}
	if drag, dragging := ui.Dragging(); dragging {
		s.dragManager.Move(drag)
	}
	drag, dragEnded := ui.DragEnd()
	if dragEnded {
		s.dragManager.End(drag)
	}

	// Keep the card positions fixed until the pointer gesture ends.
	if !ui.Pressed() && !dragEnded {
		pageChanged := s.updateDeckPage(W)
		pointer := ui.Position()
		point := image.Pt(int(float64(pointer.X)/scale), int(float64(pointer.Y)/scale))
		deckIdx, collectionIdx := s.updateCardHover(point)
		if !pageChanged {
			s.handleCardShortcuts(deckIdx, collectionIdx)
		}
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

func (s *EditDeckScreen) deckCardAt(point image.Point) int {
	start, end := s.deckPageRange()
	for i := start; i < end; i++ {
		if point.In(s.deckCardDisplays[i].visibleBounds()) {
			return i
		}
	}
	return -1
}

func (s *EditDeckScreen) updateCardHover(point image.Point) (deckIdx, collectionIdx int) {
	if i := s.deckCardAt(point); i >= 0 {
		s.magnifyCard(s.deckCardDisplays[i].Owned)
		return i, -1
	}
	buttons := s.CollectionList.GetItems()
	start := s.CollectionList.GetCurrentOffset()
	end := min(start+s.CollectionList.GetVisibleCount(), len(buttons))
	for i := start; i < end; i++ {
		if point.In(buttons[i].Bounds) {
			if group := s.collectionGroups[buttons[i].ID]; group != nil {
				s.magnifyCard(group.owned)
				return -1, i
			}
		}
	}
	return -1, -1
}

func (s *EditDeckScreen) magnifyCard(owned *domain.OwnedCard) {
	if img, err := owned.CardImage(domain.CardViewFull); err == nil {
		s.MagnifierImage = img
		s.MagnifiedCard = owned.Card
	}
}

func (s *EditDeckScreen) handleCardShortcuts(deckIdx, collectionIdx int) {
	switch {
	case collectionIdx >= 0 && inpututil.IsKeyJustPressed(ebiten.KeyA):
		s.handleDeckCardAdd(collectionIdx)
	case deckIdx >= 0 && inpututil.IsKeyJustPressed(ebiten.KeyD):
		s.handleDeckCardRemove(deckIdx)
	case deckIdx >= 0 && inpututil.IsKeyJustPressed(ebiten.KeyS):
		s.handleSellCard(deckIdx, false)
	case collectionIdx >= 0 && inpututil.IsKeyJustPressed(ebiten.KeyS):
		s.handleSellCard(collectionIdx, true)
	}
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
	for _, item := range s.deckDraggableItems {
		items = append(items, item)
	}
	return items
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

func deckPageButtonBounds(W int) (prev, next image.Rectangle) {
	back := editDeckBackBounds(W)
	return back.Sub(image.Pt(290, 0)), back.Sub(image.Pt(110, 0))
}

func (s *EditDeckScreen) drawDeckPaging(screen *ebiten.Image, W int) {
	pages := s.deckPageCount()
	if pages == 1 {
		return
	}
	prev, next := deckPageButtonBounds(W)
	if s.deckPage > 0 {
		drawDeckActionButton(screen, prev, "Prev")
	}
	if s.deckPage+1 < pages {
		drawDeckActionButton(screen, next, "Next")
	}
	pageText := elements.NewText(14, fmt.Sprintf("Page %d / %d", s.deckPage+1, pages), prev.Max.X, prev.Min.Y+6)
	pageText.HAlign = elements.AlignCenter
	pageText.BoundsW = float64(next.Min.X - prev.Max.X)
	pageText.Draw(screen, nil, 1)
}

func (s *EditDeckScreen) updateDeckPage(W int) bool {
	if s.deckPageCount() == 1 {
		return false
	}
	prev, next := deckPageButtonBounds(W)
	switch {
	case ui.Click(prev) || inpututil.IsKeyJustPressed(ebiten.KeyPageUp):
		return s.setDeckPage(s.deckPage - 1)
	case ui.Click(next) || inpututil.IsKeyJustPressed(ebiten.KeyPageDown):
		return s.setDeckPage(s.deckPage + 1)
	}
	return false
}

func (s *EditDeckScreen) setDeckPage(page int) bool {
	page = min(max(page, 0), s.deckPageCount()-1)
	if page == s.deckPage {
		return false
	}
	s.deckPage = page
	s.calculateDeckCardPositions()
	s.createDeckDraggableItems()
	s.MagnifierImage = nil
	s.MagnifiedCard = nil
	return true
}

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
	s.loadDeckCards()
	if err := s.reloadCollectionList(); err != nil {
		fmt.Printf("Error reloading collection list: %v\n", err)
	}
}

// handleDeckCardAdd adds a card from collection to the deck
func (s *EditDeckScreen) handleDeckCardAdd(cardIdx int) {
	items := s.CollectionList.GetItems()
	if cardIdx < 0 || cardIdx >= len(items) {
		return
	}
	if s.addCollectionCardToDeck(items[cardIdx].ID) {
		s.updateState()
	}
}

func (s *EditDeckScreen) addCollectionCardToDeck(name string) bool {
	group := s.collectionGroups[name]
	if group == nil || group.owned == nil || group.owned.DeckIndex != -1 {
		return false
	}
	if err := s.Player.CardCollection.MoveOwnedCard(group.owned.ID, s.Player.ActiveDeck); err != nil {
		fmt.Printf("Error moving card to deck: %v\n", err)
		return false
	}
	return true
}

// handleDeckCardRemove removes a card from deck back to collection
func (s *EditDeckScreen) handleDeckCardRemove(deckIdx int) {
	if deckIdx >= len(s.deckCardDisplays) {
		return
	}

	display := s.deckCardDisplays[deckIdx]
	cardToRemove := display.Owned.Card

	if err := s.Player.CardCollection.MoveOwnedCard(display.Owned.ID, -1); err != nil {
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
		if !exists || group.owned == nil {
			fmt.Printf("No card group found for %s\n", btn.ID)
			return
		}
		cardToSell = group.owned.Card
	} else {
		if cardIdx >= len(s.deckCardDisplays) {
			return
		}
		display := s.deckCardDisplays[cardIdx]
		cardToSell = display.Owned.Card
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
	var owned *domain.OwnedCard
	if fromDeck {
		for _, display := range s.deckCardDisplays {
			if display.Owned.Card == card {
				owned = display.Owned
				break
			}
		}
		if owned == nil {
			for _, copy := range s.Player.CardCollection.CardsInDeck(s.Player.ActiveDeck) {
				if copy.Card == card {
					owned = copy
					break
				}
			}
		}
	} else {
		if group := s.collectionGroups[card.Name()]; group != nil {
			owned = group.owned
		}
		if owned == nil {
			if copies := s.Player.CardCollection.AvailableCards(card); len(copies) > 0 {
				owned = copies[0]
			}
		}
	}
	if owned == nil {
		return false
	}
	salePrice := owned.SalePrice(s.City)
	if err := s.Player.CardCollection.RemoveOwnedCard(owned.ID); err != nil {
		return false
	}
	s.Player.Gold += salePrice
	fmt.Printf("Sold %s for %d gold\n", card.Name(), salePrice)
	return true
}

// loadDeckCards loads the player's deck cards for display
func (s *EditDeckScreen) loadDeckCards() {
	s.deckCardDisplays = make([]DeckCardDisplay, 0)

	deckGroups := make(map[string]*cardGroup)
	for _, owned := range s.Player.CardCollection.CardsInDeck(s.Player.ActiveDeck) {
		if group := deckGroups[owned.Name()]; group != nil {
			group.totalCount++
		} else {
			deckGroups[owned.Name()] = &cardGroup{owned: owned, totalCount: 1}
		}
	}

	for _, group := range deckGroups {
		img, err := group.owned.CardImage(domain.CardViewArtMini)
		if err != nil || img == nil {
			continue
		}
		s.deckCardDisplays = append(s.deckCardDisplays, DeckCardDisplay{
			Owned:     group.owned,
			Image:     img,
			Count:     group.totalCount,
			colorRank: deckEditorColorRank(group.owned.ColorMask()),
			manaValue: group.owned.ManaValue(),
		})
	}
	slices.SortFunc(s.deckCardDisplays, func(a, b DeckCardDisplay) int {
		if byColor := cmp.Compare(a.colorRank, b.colorRank); byColor != 0 {
			return byColor
		}
		if byMana := cmp.Compare(a.manaValue, b.manaValue); byMana != 0 {
			return byMana
		}
		return cmp.Compare(a.Owned.Name(), b.Owned.Name())
	})

	// Calculate positions for deck cards
	s.calculateDeckCardPositions()

	// Create draggable items for deck cards
	s.createDeckDraggableItems()
}

func deckEditorColorRank(colors domain.ColorMask) int {
	const multicolor = int(domain.ColorGreen) + 1
	switch colors {
	case domain.ColorWhite, domain.ColorBlue, domain.ColorBlack, domain.ColorRed, domain.ColorGreen:
		return int(colors)
	case domain.ColorColorless:
		return multicolor + 1
	default:
		return multicolor
	}
}

func (s *EditDeckScreen) deckPageCount() int {
	pageSize := max(s.deckPageSize, 1)
	return max(1, (len(s.deckCardDisplays)+pageSize-1)/pageSize)
}

func (s *EditDeckScreen) deckPageRange() (start, end int) {
	start = s.deckPage * s.deckPageSize
	end = min(start+s.deckPageSize, len(s.deckCardDisplays))
	return start, end
}

func (s *EditDeckScreen) calculateDeckCardPositions() {
	bounds := s.deckDropArea.GetDropBounds()
	const columnStep = domain.CardArtMiniWidth + deckColumnGap
	cols := max(1, (bounds.Dx()-2*deckSideInset+deckColumnGap)/columnStep)
	// Leave space for the full height of the last row.
	rowSpace := max(0, bounds.Dy()-deckTopInset-domain.CardArtMiniHeight)
	s.deckPageSize = cols * (1 + rowSpace/deckMinRowStep)
	s.deckPage = min(s.deckPage, s.deckPageCount()-1)

	start, end := s.deckPageRange()
	rowIntervals := max(1, (end-start-1)/cols)
	rowStep := min(domain.CardArtMiniHeight+deckRowGap, rowSpace/rowIntervals)
	for i := start; i < end; i++ {
		col := (i - start) % cols
		row := (i - start) / cols
		display := &s.deckCardDisplays[i]
		display.X = bounds.Min.X + deckSideInset + col*columnStep
		display.Y = bounds.Min.Y + deckTopInset + row*rowStep
		display.visibleHeight = domain.CardArtMiniHeight
		if i+cols < end {
			display.visibleHeight = min(display.visibleHeight, rowStep)
		}
	}
}

// createDeckDraggableItems creates draggable buttons for deck cards
func (s *EditDeckScreen) createDeckDraggableItems() {
	start, end := s.deckPageRange()
	s.deckDraggableItems = make([]*dragdrop.DraggableButton, 0, end-start)

	for i := start; i < end; i++ {
		display := &s.deckCardDisplays[i]
		img := display.cardImage()
		if img == nil {
			continue
		}

		btn := elements.NewButton(img, img, img, display.X, display.Y, 1.0)
		btn.Bounds = display.visibleBounds()
		btn.ID = deckCardDragIDPrefix + display.Owned.Name()

		draggableBtn := dragdrop.NewDraggableButton(btn, display.Owned.Card)
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

	return s.addCollectionCardToDeck(droppedCard.Name())
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

	var err error
	var owned *domain.OwnedCard
	for _, display := range s.deckCardDisplays {
		if display.Owned.Card == droppedCard {
			owned = display.Owned
			break
		}
	}
	if owned != nil {
		err = s.Player.CardCollection.MoveOwnedCard(owned.ID, -1)
	} else {
		err = s.Player.CardCollection.MoveCardFromDeck(droppedCard, s.Player.ActiveDeck, 1)
	}
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

		if display.Owned.ManaCost != "" {
			cmc := min(display.manaValue, 7)
			cmcCounts[cmc] += display.Count
		}

		for _, c := range display.Owned.Colors {
			colorCounts[c] += display.Count
		}

		switch display.Owned.CardType {
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
