package screens

import (
	"fmt"
	"image"
	"image/color"

	"github.com/benprew/s30/assets"
	gameaudio "github.com/benprew/s30/game/audio"
	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/ui/elements"
	"github.com/benprew/s30/game/ui/fonts"
	"github.com/benprew/s30/game/ui/imageutil"
	"github.com/benprew/s30/game/ui/screenui"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// Dungeon sprite-sheet layout. Sheets are 12 columns × N rows (N >= 2) of
// 64×64 cells. The floor diamond sits in the lower portion of each cell
// (roughly y=24..y=56), with walls extending up into the top portion.
//
//	row 1 col 0    → plain floor diamond
//	row 0 col 8    → treasure chest overlay
//	row 0 col 9    → scroll / trivia overlay
//	row 0 col 10   → dice overlay
//	row 1 col 8    → rocky bedrock (unwalkable)
const (
	dungeonSheetCols    = 12
	dungeonCellPixels   = 64
	dungeonTileW        = 64 // diamond width in screen pixels
	dungeonTileH        = 32 // diamond height in screen pixels
	dungeonCellTreasure = 8
	dungeonCellScroll   = 9
	dungeonCellDice     = 10
	dungeonCellWall     = 8
	dungeonFloorRow     = 1
	dungeonFloorCol     = 0
)

// dungeonCharScale shrinks the 248×174 walking-sprite cells down to fit on a
// 64-wide diamond. Tuned by eye against dungeon_ex.png.
const dungeonCharScale = 0.6

// charAnchor{X,Y} is the figure's feet position within a 248×174 walking
// sheet cell. Measured from the non-transparent bbox in Ego_F.spr.png across
// frames: figure body sits at roughly x=99..120, y=45..102 with ~70 px of
// whitespace below. Anchoring at the feet pins the character to the floor
// diamond instead of leaving it floating above.
const (
	charAnchorX = 110.0
	charAnchorY = 80.0
)

type characterFrames struct {
	sprite [][]*ebiten.Image
	shadow [][]*ebiten.Image
}

type DungeonScreen struct {
	Player        *domain.Player
	Level         *world.Level
	sheet         [][]*ebiten.Image
	playerFrames  characterFrames
	enemyFrames   map[*domain.Character]characterFrames
	enemyFallback *ebiten.Image
	gridLeft      int
	gridTop       int
	overlayActive bool
	overlayTile   image.Point
	overlayBtns   []*elements.Button
	overlayTitle  string
	overlayBody   string
}

func NewDungeonScreen(player *domain.Player, level *world.Level) *DungeonScreen {
	s := &DungeonScreen{
		Player: player,
		Level:  level,
	}

	s.enemyFrames = make(map[*domain.Character]characterFrames)
	s.enemyFallback = makeColorTile(color.RGBA{R: 200, G: 60, B: 60, A: 255}, dungeonCellPixels-24)

	if st := player.DungeonState; st != nil && st.CurrentDungeon != nil {
		s.sheet = loadDungeonSheet(st.CurrentDungeon.Color)
		W := st.CurrentDungeon.Width()
		H := st.CurrentDungeon.Height()
		// Iso bounding box of the full sprite footprint (cells overlap).
		totalW := (W+H-2)*dungeonTileW/2 + dungeonCellPixels
		totalH := (W+H-2)*dungeonTileH/2 + dungeonCellPixels
		s.gridLeft = (1024-totalW)/2 + (H-1)*dungeonTileW/2
		s.gridTop = (768 - totalH) / 2

		s.cacheEnemySprites(st.CurrentDungeon)
	}

	s.playerFrames = scaleCharacterFrames(player.WalkingSprite, player.ShadowSprite, dungeonCharScale)
	return s
}

// cacheEnemySprites pre-scales each unique enemy character's walking sheet so
// Draw() never resizes images per frame.
func (s *DungeonScreen) cacheEnemySprites(d *domain.Dungeon) {
	for y := 0; y < d.Height(); y++ {
		for x := 0; x < d.Width(); x++ {
			tile := &d.Grid[y][x]
			if tile.Type != domain.DungeonTileEnemy || tile.Enemy == nil {
				continue
			}
			if _, ok := s.enemyFrames[tile.Enemy]; ok {
				continue
			}
			if tile.Enemy.WalkingSprite == nil {
				_ = tile.Enemy.LoadImages()
			}
			s.enemyFrames[tile.Enemy] = scaleCharacterFrames(
				tile.Enemy.WalkingSprite, tile.Enemy.ShadowSprite, dungeonCharScale,
			)
		}
	}
}

// scaleCharacterFrames returns scaled-down copies of an [direction][frame]
// sprite sheet plus its matching shadow sheet. Either sheet may be nil.
func scaleCharacterFrames(sprite, shadow [][]*ebiten.Image, scale float64) characterFrames {
	return characterFrames{
		sprite: scaleSheet(sprite, scale),
		shadow: scaleSheet(shadow, scale),
	}
}

func scaleSheet(sheet [][]*ebiten.Image, scale float64) [][]*ebiten.Image {
	if sheet == nil {
		return nil
	}
	out := make([][]*ebiten.Image, len(sheet))
	for i, row := range sheet {
		out[i] = make([]*ebiten.Image, len(row))
		for j, src := range row {
			if src == nil {
				continue
			}
			b := src.Bounds()
			w := int(float64(b.Dx())*scale + 0.5)
			h := int(float64(b.Dy())*scale + 0.5)
			if w <= 0 || h <= 0 {
				continue
			}
			dst := ebiten.NewImage(w, h)
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Scale(scale, scale)
			dst.DrawImage(src, opts)
			out[i][j] = dst
		}
	}
	return out
}

func (s *DungeonScreen) IsFramed() bool { return false }

func (s *DungeonScreen) dungeonState() *domain.DungeonState {
	return s.Player.DungeonState
}

func (s *DungeonScreen) Update(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	st := s.dungeonState()
	if st == nil || st.CurrentDungeon == nil {
		return screenui.WorldScr, nil, nil
	}

	if s.overlayActive {
		return s.updateOverlay(W, H, scale)
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return s.exitDungeon(), nil, nil
	}

	dx, dy := 0, 0
	switch {
	case inpututil.IsKeyJustPressed(ebiten.KeyUp), inpututil.IsKeyJustPressed(ebiten.KeyW):
		dy = -1
		s.Player.Direction = domain.DirectionUpRight
	case inpututil.IsKeyJustPressed(ebiten.KeyDown), inpututil.IsKeyJustPressed(ebiten.KeyS):
		dy = 1
		s.Player.Direction = domain.DirectionDownLeft
	case inpututil.IsKeyJustPressed(ebiten.KeyLeft), inpututil.IsKeyJustPressed(ebiten.KeyA):
		dx = -1
		s.Player.Direction = domain.DirectionUpLeft
	case inpututil.IsKeyJustPressed(ebiten.KeyRight), inpututil.IsKeyJustPressed(ebiten.KeyD):
		dx = 1
		s.Player.Direction = domain.DirectionDownRight
	}

	if dx == 0 && dy == 0 {
		return screenui.DungeonScr, nil, nil
	}

	target := image.Point{X: st.Position.X + dx, Y: st.Position.Y + dy}
	dungeon := st.CurrentDungeon
	tile := dungeon.Tile(target)
	if tile == nil || !tile.IsWalkable() {
		return screenui.DungeonScr, nil, nil
	}

	st.Position = target
	s.Player.Frame = (s.Player.Frame + 1) % domain.SpriteCols
	tile.Visited = true
	dungeon.RevealFrom(target)

	if tile.Type == domain.DungeonTileEnemy && tile.Enemy != nil {
		return s.startDungeonDuel(tile)
	}

	if tile.Type == domain.DungeonTileTreasure && tile.Reward != nil {
		s.openTreasureOverlay(target)
	}

	if tile.Type == domain.DungeonTileDice && tile.Dice != nil {
		s.openDiceOverlay(target)
	}
	return screenui.DungeonScr, nil, nil
}

// startDungeonDuel begins a duel against the rogue on `tile`. Dungeon duels are
// immediate: there is no ante screen and no bribe option.
func (s *DungeonScreen) startDungeonDuel(tile *domain.DungeonTile) (screenui.ScreenName, screenui.Screen, error) {
	if am := gameaudio.Get(); am != nil {
		am.PlaySFX(gameaudio.EnemySFXForName(tile.Enemy.Name))
	}
	enemy := domain.NewEnemyFromCharacter(tile.Enemy)
	duel := NewDungeonDuelScreen(s.Player, &enemy, s.dungeonState(), tile)
	return screenui.DuelScr, duel, nil
}

func (s *DungeonScreen) exitDungeon() screenui.ScreenName {
	s.Player.ExitDungeon()
	return screenui.WorldScr
}

func (s *DungeonScreen) openTreasureOverlay(p image.Point) {
	st := s.dungeonState()
	s.overlayActive = true
	s.overlayTile = p
	s.overlayTitle = "Treasure!"
	s.overlayBody = rewardDescriptionText(st.CurrentDungeon.Tile(p).Reward)
	s.overlayBtns = makeOverlayButtons("Take", "Leave")
}

// openDiceOverlay applies the dice tile at p immediately (a dice roll cannot be
// refused) and shows an informational overlay describing the result.
func (s *DungeonScreen) openDiceOverlay(p image.Point) {
	st := s.dungeonState()
	desc := st.ApplyDice(st.CurrentDungeon.Tile(p), s.Player)
	if desc == "" {
		return
	}
	s.overlayActive = true
	s.overlayTile = p
	s.overlayTitle = "The Dice are Cast!"
	s.overlayBody = desc
	s.overlayBtns = []*elements.Button{makeOverlayButton("Continue", buttonIDLeave, 512)}
}

func (s *DungeonScreen) updateOverlay(W, H int, scale float64) (screenui.ScreenName, screenui.Screen, error) {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		s.closeOverlay()
		return screenui.DungeonScr, nil, nil
	}

	opts := &ebiten.DrawImageOptions{}
	for _, b := range s.overlayBtns {
		b.Update(opts, scale, W, H)
		if !b.IsClicked() {
			continue
		}
		if b.ID == "take" {
			s.collectCurrentReward()
		}
		s.closeOverlay()
		return screenui.DungeonScr, nil, nil
	}
	return screenui.DungeonScr, nil, nil
}

func (s *DungeonScreen) collectCurrentReward() {
	st := s.dungeonState()
	if st == nil {
		return
	}
	st.CollectReward(st.CurrentDungeon.Tile(s.overlayTile), s.Player)
}

func (s *DungeonScreen) closeOverlay() {
	s.overlayActive = false
	s.overlayBtns = nil
	s.overlayTitle = ""
	s.overlayBody = ""
}

func (s *DungeonScreen) Draw(screen *ebiten.Image, W, H int, scale float64) {
	screen.Fill(color.RGBA{R: 8, G: 6, B: 14, A: 255})

	st := s.dungeonState()
	if st == nil || st.CurrentDungeon == nil {
		return
	}
	dungeon := st.CurrentDungeon

	// Painter's order: iterate diagonals so back tiles draw before front tiles,
	// and the player draws on top of its own tile but behind tiles in front.
	dw, dh := dungeon.Width(), dungeon.Height()
	for sum := 0; sum <= dw+dh-2; sum++ {
		for gy := 0; gy <= sum; gy++ {
			gx := sum - gy
			if gx < 0 || gx >= dw || gy >= dh {
				continue
			}
			s.drawTile(screen, gx, gy, &dungeon.Grid[gy][gx])
			if st.Position.X == gx && st.Position.Y == gy {
				s.drawCharacter(screen, st.Position, s.playerFrames, s.Player.Direction, s.Player.Frame)
			}
		}
	}

	s.drawHUD(screen, dungeon, scale)

	if s.overlayActive {
		s.drawOverlay(screen, scale)
	}
}

func (s *DungeonScreen) drawTile(screen *ebiten.Image, x, y int, tile *domain.DungeonTile) {
	if !tile.Seen {
		return
	}

	sx, sy := s.tileScreenPos(x, y)
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(sx), float64(sy))

	floor, overlay := s.spriteFor(tile)
	if floor != nil {
		screen.DrawImage(floor, opts)
	}
	if overlay != nil {
		screen.DrawImage(overlay, opts)
	}

	if tile.Type == domain.DungeonTileEnemy {
		s.drawEnemy(screen, image.Point{X: x, Y: y}, tile.Enemy)
	}
}

// tileScreenPos returns the on-screen top-left of the 64×64 sprite cell that
// renders dungeon tile (x, y). Cells overlap in iso layout; the diamond floor
// itself sits in the lower portion of the cell.
func (s *DungeonScreen) tileScreenPos(x, y int) (int, int) {
	return s.gridLeft + (x-y)*dungeonTileW/2, s.gridTop + (x+y)*dungeonTileH/2
}

// spriteFor returns the floor sprite and an optional overlay (chest, scroll,
// dice) to stack on top. Walls render the rocky bedrock in place of a floor.
func (s *DungeonScreen) spriteFor(tile *domain.DungeonTile) (*ebiten.Image, *ebiten.Image) {
	if s.sheet == nil {
		return nil, nil
	}
	floor := s.cell(dungeonFloorRow, dungeonFloorCol)
	switch tile.Type {
	case domain.DungeonTileWall:
		return s.cell(1, dungeonCellWall), nil
	case domain.DungeonTileTreasure:
		return floor, s.cell(0, dungeonCellTreasure)
	case domain.DungeonTileScroll:
		return floor, s.cell(0, dungeonCellScroll)
	case domain.DungeonTileDice:
		return floor, s.cell(0, dungeonCellDice)
	default:
		return floor, nil
	}
}

func (s *DungeonScreen) cell(row, col int) *ebiten.Image {
	if row < 0 || row >= len(s.sheet) {
		return nil
	}
	if col < 0 || col >= len(s.sheet[row]) {
		return nil
	}
	return s.sheet[row][col]
}

// drawCharacter renders a walking-sprite character on the diamond floor of
// tile p. The floor diamond center sits at (32, 40) within each 64×64 cell;
// the character's feet line up with that point.
func (s *DungeonScreen) drawCharacter(screen *ebiten.Image, p image.Point, frames characterFrames, dir, frame int) {
	sprite := frameAt(frames.sprite, dir, frame)
	if sprite == nil {
		return
	}
	shadow := frameAt(frames.shadow, dir, frame)

	sx, sy := s.tileScreenPos(p.X, p.Y)
	cx := float64(sx + dungeonCellPixels/2)
	cy := float64(sy + 40)

	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(-charAnchorX*dungeonCharScale, -charAnchorY*dungeonCharScale)
	opts.GeoM.Translate(cx, cy)

	if shadow != nil {
		screen.DrawImage(shadow, opts)
	}
	screen.DrawImage(sprite, opts)
}

func frameAt(sheet [][]*ebiten.Image, dir, frame int) *ebiten.Image {
	if dir < 0 || dir >= len(sheet) {
		return nil
	}
	row := sheet[dir]
	if frame < 0 || frame >= len(row) {
		return nil
	}
	return row[frame]
}

// drawEnemy renders an enemy character at tile p, falling back to a flat red
// square when the enemy has no walking sheet (e.g. a dungeon generated without
// an enemy pool).
func (s *DungeonScreen) drawEnemy(screen *ebiten.Image, p image.Point, enemy *domain.Character) {
	if enemy != nil {
		if frames, ok := s.enemyFrames[enemy]; ok {
			s.drawCharacter(screen, p, frames, domain.DirectionDown, 0)
			return
		}
	}
	if s.enemyFallback == nil {
		return
	}
	sx, sy := s.tileScreenPos(p.X, p.Y)
	cx := sx + dungeonCellPixels/2
	cy := sy + 40
	w := s.enemyFallback.Bounds().Dx()
	h := s.enemyFallback.Bounds().Dy()
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Translate(float64(cx-w/2), float64(cy-h))
	screen.DrawImage(s.enemyFallback, opts)
}

func (s *DungeonScreen) drawHUD(screen *ebiten.Image, dungeon *domain.Dungeon, scale float64) {
	name := elements.NewText(22, dungeon.Name, 30, 30)
	name.Color = color.White
	name.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	st := s.dungeonState()
	statsText := fmt.Sprintf("Life: %d   Cards collected: %d", st.DungeonLife, len(st.CollectedCards))
	if bonus := s.Player.BonusDuelLife; bonus != 0 {
		statsText += fmt.Sprintf("   Dice life: %+d", bonus)
	}
	if len(s.Player.BonusDuelCards) > 0 {
		statsText += fmt.Sprintf("   Dice card: %s", s.Player.BonusDuelCards[len(s.Player.BonusDuelCards)-1].CardName)
	}
	stats := elements.NewText(18, statsText, 30, 65)
	stats.Color = color.RGBA{R: 220, G: 220, B: 240, A: 255}
	stats.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	hint := elements.NewText(16, "Arrow keys / WASD to move    ESC to leave", 30, 728)
	hint.Color = color.RGBA{R: 160, G: 160, B: 180, A: 255}
	hint.Draw(screen, &ebiten.DrawImageOptions{}, scale)
}

func (s *DungeonScreen) drawOverlay(screen *ebiten.Image, scale float64) {
	dim := ebiten.NewImage(1024, 768)
	dim.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 180})
	screen.DrawImage(dim, &ebiten.DrawImageOptions{})

	panelW, panelH := 520, 280
	px := (1024 - panelW) / 2
	py := (768 - panelH) / 2
	panel := ebiten.NewImage(panelW, panelH)
	panel.Fill(color.RGBA{R: 30, G: 25, B: 50, A: 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(px), float64(py))
	screen.DrawImage(panel, op)

	title := elements.NewText(28, s.overlayTitle, px+30, py+30)
	title.Color = color.White
	title.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	body := elements.NewText(20, s.overlayBody, px+30, py+90)
	body.Color = color.RGBA{R: 230, G: 230, B: 240, A: 255}
	body.Draw(screen, &ebiten.DrawImageOptions{}, scale)

	for _, b := range s.overlayBtns {
		b.Draw(screen, &ebiten.DrawImageOptions{}, scale)
	}
}

func rewardDescriptionText(r *domain.DungeonReward) string {
	switch r.Type {
	case domain.DungeonRewardRestrictedCard:
		if r.Card != nil {
			return fmt.Sprintf("A restricted card: %s", r.Card.CardName)
		}
		return "A restricted card."
	case domain.DungeonRewardGoldAmulets:
		if len(r.Amulets) == 0 {
			return fmt.Sprintf("%d gold pieces.", r.Gold)
		}
		return fmt.Sprintf("%d gold and %d amulet(s).", r.Gold, len(r.Amulets))
	}
	return "Something glints in the chest."
}

// loadDungeonSheet returns the 12-column sprite sheet for the dungeon's color.
// Falls back to white if the color does not have a dedicated sheet.
func loadDungeonSheet(c domain.ColorMask) [][]*ebiten.Image {
	bytes := dungeonSheetBytesFor(c)
	rows := dungeonSheetRowsFor(c)
	sheet, err := imageutil.LoadSpriteSheet(dungeonSheetCols, rows, bytes)
	if err != nil {
		panic(fmt.Sprintf("dungeon sheet load: %v", err))
	}
	return sheet
}

func dungeonSheetBytesFor(c domain.ColorMask) []byte {
	switch c {
	case domain.ColorWhite:
		return assets.DungeonW_png
	case domain.ColorBlue:
		return assets.DungeonU_png
	case domain.ColorBlack:
		return assets.DungeonB_png
	case domain.ColorRed:
		return assets.DungeonR_png
	case domain.ColorGreen:
		return assets.DungeonG_png
	default:
		return assets.DungeonW_png
	}
}

// dungeonSheetRowsFor reports the number of 64-px rows in the sprite sheet
// for color c. The colored sheets are 320 px tall (5 rows); colorless / fallback
// uses the same.
func dungeonSheetRowsFor(_ domain.ColorMask) int {
	return 5
}

func makeColorTile(c color.Color, size int) *ebiten.Image {
	img := ebiten.NewImage(size, size)
	img.Fill(c)
	return img
}

func makeOverlayButtons(takeText, leaveText string) []*elements.Button {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	font := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}

	takeW, _ := elements.TextButtonSize(takeText, font)
	leaveW, _ := elements.TextButtonSize(leaveText, font)
	totalW := takeW + 20 + leaveW
	startX := 512 - totalW/2

	take := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: takeText, Font: font, ID: "take",
		X: startX, Y: 460,
	})
	leave := elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: leaveText, Font: font, ID: buttonIDLeave,
		X: startX + takeW + 20, Y: 460,
	})
	return []*elements.Button{take, leave}
}

// makeOverlayButton builds a single overlay button horizontally centered on
// centerX.
func makeOverlayButton(label, id string, centerX int) *elements.Button {
	btnSprites, err := imageutil.LoadSpriteSheet(3, 1, assets.Tradbut1_png)
	if err != nil {
		panic(err)
	}
	font := &text.GoTextFace{Source: fonts.MtgFont, Size: 18}
	w, _ := elements.TextButtonSize(label, font)
	return elements.NewButtonFromConfig(elements.ButtonConfig{
		Normal: btnSprites[0][0], Hover: btnSprites[0][1], Pressed: btnSprites[0][2],
		Text: label, Font: font, ID: id,
		X: centerX - w/2, Y: 460,
	})
}
