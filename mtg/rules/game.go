package mtg

type GameState struct {
	Players     []*Player
	Battlefield []*Card
	Graveyard   []*Card
	Exile       []*Card
	Turn        *Turn
}

type Player struct {
	LifeTotal int
	ManaPool  *ManaPool
	Hand      []*Card
	Library   []*Card
}

func NewGame(players []*Player) *GameState {
	return &GameState{
		Players: players,
		Turn: &Turn{
			Phase: PhaseUntap,
		},
	}
}

func (g *GameState) CheckWinConditions() {
	for _, player := range g.Players {
		if player.LifeTotal <= 0 {
			// Player lost
			// TODO: Implement game over logic
		}
		if len(player.Library) == 0 {
			// Player lost due to running out of cards
			// TODO: Implement game over logic
		}
	}
}

func (g *GameState) StartGame() {
	// TODO: Implement game start logic (draw hands, etc.)
}

func (g *GameState) NextTurn() {
	g.Turn.NextPhase()
	if g.Turn.Phase == PhaseUntap {
		// Start of new turn
		// TODO: Implement untap logic
		// TODO: Implement upkeep logic
		// TODO: Implement draw logic
	}
}
