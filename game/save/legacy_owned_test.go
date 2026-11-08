package save

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func TestLegacySaveMigratesCardsAndKeepsThemAfterResave(t *testing.T) {
	for _, version := range []int{1, 2} {
		t.Run(fmt.Sprintf("version_%d", version), func(t *testing.T) {
			collection := `[
				{"card_id":"atq-80b-mishras-factory","card_name":"Mishra's Factory","count":3,"deck_counts":[1,1,0]},
				{"card_id":"atq-80d-mishras-factory","card_name":"Mishra's Factory","count":2,"deck_counts":[1,0,0]},
				{"card_name":"Lightning Bolt","count":1,"deck_counts":[0,0,0]}
			]`
			if version == 2 {
				collection = `[
					{"card_id":"atq-80c-mishras-factory","count":2,"deck_counts":[1,1,0],"instances":[
						{"id":"saved-copy-1","card":{"CardID":"atq-80c-mishras-factory"},"artwork_id":"atq-80b-mishras-factory","deck_index":0},
						{"id":"saved-copy-2","card":{"CardID":"atq-80c-mishras-factory"},"artwork_id":"atq-80d-mishras-factory","deck_index":1}
					]}
				]`
			}
			data := []byte(fmt.Sprintf(`{
				"version":%d,"game_id":"legacy-game","world":{
					"GameID":"legacy-game","W":1,"H":1,
					"Player":{"Gold":543,"Food":9,"Life":15,"X":100,"Y":200,"ActiveDeck":1,
						"CardCollection":%s,"BonusDuelCards":[{"CardID":"4ed-208-lightning-bolt"}]},
					"Tiles":[[{"City":{"Name":"Old Town","CardsForSale":[
						{"CardID":"atq-80a-mishras-factory"},
						{"CardName":"Mishra's Factory","SetID":"atq","CollectorNo":"80d","PngURL":"stale-url"}
					]}}]]
				}
			}`, version, collection))
			loaded, err := deserializeSave(data)
			if err != nil {
				t.Fatal(err)
			}
			player := loaded.World.Player
			factory := domain.FindCardByName("Mishra's Factory")
			if player.Gold != 543 || player.Food != 9 || player.Life != 15 || player.X != 100 || player.Y != 200 || player.ActiveDeck != 1 {
				t.Fatal("loading changed the player's resources or position")
			}
			if player.GetNumDecks() != 3 {
				t.Fatalf("deck slots = %d, want 3", player.GetNumDecks())
			}
			counts := map[string]int{}
			for _, owned := range player.CardCollection[factory].Instances {
				if owned.ID == "" || owned.Card != factory {
					t.Fatal("migration did not create a canonical owned copy")
				}
				counts[owned.ArtworkID]++
				if owned.Artwork().ID != owned.ArtworkID {
					t.Fatal("migration lost a known printing")
				}
			}
			if version == 1 {
				if counts["atq-80b-mishras-factory"] != 3 || counts["atq-80d-mishras-factory"] != 2 || player.CardCollection.NumCards() != 6 {
					t.Fatalf("migration changed card counts: %v", counts)
				}
				if player.CardCollection.GetDeckCount(factory, 0) != 2 || player.CardCollection.GetDeckCount(factory, 1) != 1 || len(player.CardCollection.AvailableCards(factory)) != 2 {
					t.Fatal("migration changed deck membership")
				}
			} else {
				if player.CardCollection.CardsInDeck(0)[0].ID != "saved-copy-1" || player.CardCollection.CardsInDeck(1)[0].ID != "saved-copy-2" {
					t.Fatal("version 2 lost existing owned-copy IDs")
				}
			}
			shop := loaded.World.Tiles[0][0].City.CardsForSale
			if len(shop) != 2 || shop[0].ArtworkID != "atq-80a-mishras-factory" || shop[1].ArtworkID != "atq-80d-mishras-factory" {
				t.Fatal("migration lost shop stock or its artwork")
			}
			if len(player.BonusDuelCards) != 1 || player.BonusDuelCards[0].Name() != "Lightning Bolt" {
				t.Fatal("migration lost the bonus card")
			}
			before := ownedSaveSnapshot(t, loaded.World)
			updated, err := serializeSave(loaded.World)
			if err != nil {
				t.Fatal(err)
			}
			reloaded, err := deserializeSave(updated)
			if err != nil {
				t.Fatal(err)
			}
			if reloaded.Version != currentSaveVersion {
				t.Fatalf("resave version = %d", reloaded.Version)
			}
			if reloaded.World.Player.GetNumDecks() != 3 {
				t.Fatal("resaving lost an empty deck slot")
			}
			if !reflect.DeepEqual(before, ownedSaveSnapshot(t, reloaded.World)) {
				t.Fatal("saving and reloading changed owned copies")
			}
		})
	}
}

func ownedSaveSnapshot(t *testing.T, level *world.Level) map[string]string {
	t.Helper()
	state := make(map[string]string)
	add := func(location string, owned *domain.OwnedCard) {
		if owned == nil || owned.Card == nil || owned.ID == "" {
			t.Fatalf("invalid owned card at %s", location)
		}
		if _, duplicate := state[owned.ID]; duplicate {
			t.Fatalf("duplicate owned ID %s", owned.ID)
		}
		state[owned.ID] = fmt.Sprintf("%s|%s|%s|%d", location, owned.Name(), owned.ArtworkID, owned.DeckIndex)
	}
	collection := func(location string, cards domain.CardCollection) {
		for _, item := range cards {
			for _, owned := range item.Instances {
				add(location, owned)
			}
		}
	}
	collection("player", level.Player.CardCollection)
	for y, row := range level.Tiles {
		for x, tile := range row {
			if tile == nil || tile.City == nil {
				continue
			}
			for i, owned := range tile.City.CardsForSale {
				add(fmt.Sprintf("shop-%d-%d-%d", x, y, i), owned)
			}
		}
	}
	return state
}

func TestNestedLegacyCardRetainsItsPrinting(t *testing.T) {
	var owned domain.OwnedCard
	if err := json.Unmarshal([]byte(`{"id":"old-copy","card":{"CardName":"Mishra's Factory","SetID":"atq","CollectorNo":"80b"},"deck_index":0}`), &owned); err != nil {
		t.Fatal(err)
	}
	if owned.ArtworkID != "atq-80b-mishras-factory" {
		t.Fatalf("artwork = %q, want saved printing", owned.ArtworkID)
	}
}
