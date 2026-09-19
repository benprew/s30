package duel

import (
	"bytes"
	"image"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
	"github.com/klauspost/compress/zstd"
)

func TestStartDuel_WinMoves3CardsToPlayerCollection(t *testing.T) {
	mountain := domain.FindCardByName("Mountain")
	lightningBolt := domain.FindCardByName("Lightning Bolt")

	playerDeck := make(domain.Deck)
	playerDeck[mountain] = 4
	playerDeck[lightningBolt] = 4

	playerCollection := domain.NewCardCollection()
	for card, count := range playerDeck {
		playerCollection.AddCardToDeck(card, 0, count)
	}

	player := &domain.Player{
		Character: domain.Character{
			CardCollection: playerCollection,
		},
		Amulets: make(map[domain.ColorMask]int),
	}

	forest := domain.FindCardByName("Forest")
	llanowarElves := domain.FindCardByName("Llanowar Elves")
	giantGrowth := domain.FindCardByName("Giant Growth")
	thicketBasilisk := domain.FindCardByName("Thicket Basilisk")
	crawWurm := domain.FindCardByName("Craw Wurm")

	enemyDeck := make(domain.Deck)
	enemyDeck[forest] = 4
	enemyDeck[llanowarElves] = 2
	enemyDeck[giantGrowth] = 3
	enemyDeck[thicketBasilisk] = 1
	enemyDeck[crawWurm] = 2

	enemyCollection := domain.NewCardCollection()
	for card, count := range enemyDeck {
		enemyCollection.AddCardToDeck(card, 0, count)
	}

	enemyCharacter := &domain.Character{
		CardCollection: enemyCollection,
	}

	enemy := &domain.Enemy{
		Character: enemyCharacter,
	}

	lvl := &world.Level{
		Player: player,
	}

	screen := &DuelAnteScreen{
		player:         player,
		enemy:          enemy,
		lvl:            lvl,
		idx:            0,
		playerAnteCard: mountain,
	}

	screen.startDuel()

	playerCollectionSize := 0
	for _, item := range player.CardCollection {
		playerCollectionSize += item.Count
	}

	if playerCollectionSize == 0 {
		t.Errorf("Expected player to have won cards, but collection is empty")
	}
}

func TestAnteImagesRefreshAfterArtworkArrives(t *testing.T) {
	domain.ClearCardImageCache()
	t.Cleanup(domain.ClearCardImageCache)
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatal(err)
	}
	defer encoder.Close()
	cards := domain.LoadCardDatabase(bytes.NewReader(encoder.EncodeAll([]byte(`[
 {"CardName":"Ante Test Player","SetID":"test","CollectorNo":"1"},
 {"CardName":"Ante Test Enemy","SetID":"test","CollectorNo":"2"}
 ]`), nil)))
	playerCard, enemyCard := *cards[0], *cards[1]
	for _, card := range cards {
		domain.CacheCardImage(card.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	}
	playerPlaceholder, err := playerCard.CardImage(domain.CardViewFullMini)
	if err != nil {
		t.Fatal(err)
	}
	enemyPlaceholder, err := enemyCard.CardImage(domain.CardViewFullMini)
	if err != nil {
		t.Fatal(err)
	}
	s := &DuelAnteScreen{
		playerAnteCard: &playerCard,
		enemyAnteCard:  &enemyCard,
	}
	domain.CacheCardImage(playerCard.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	playerImg, enemyImg := s.cardImages()
	playerArt, _ := playerCard.CardImage(domain.CardViewFullMini)
	if playerImg == playerPlaceholder || playerImg != playerArt {
		t.Fatal("player ante retained its placeholder")
	}
	if enemyImg != enemyPlaceholder {
		t.Fatal("enemy placeholder changed before art arrived")
	}
	domain.CacheCardImage(enemyCard.CardID(), image.NewRGBA(image.Rect(0, 0, 245, 342)))
	playerImg, enemyImg = s.cardImages()
	enemyArt, _ := enemyCard.CardImage(domain.CardViewFullMini)
	if enemyImg == enemyPlaceholder || enemyImg != enemyArt {
		t.Fatal("enemy ante retained its placeholder")
	}
	if playerImg != playerArt {
		t.Fatal("player artwork was recreated")
	}
	if enemyImg.Bounds().Size() != image.Pt(183, 256) {
		t.Fatal("ante image has the wrong size")
	}
}
