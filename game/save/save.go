package save

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/game/world"
)

func SaveGame(player *domain.Player, level *world.Level, saveName string) (string, error) {
	saveData := &SaveData{
		Version: 1,
		SavedAt: time.Now(),
	}

	var err error
	saveData.Player, err = serializePlayer(player)
	if err != nil {
		return "", fmt.Errorf("failed to serialize player: %w", err)
	}

	saveData.World, err = serializeWorld(level)
	if err != nil {
		return "", fmt.Errorf("failed to serialize world: %w", err)
	}

	saveData.Enemies, err = serializeEnemies(level.Enemies())
	if err != nil {
		return "", fmt.Errorf("failed to serialize enemies: %w", err)
	}

	jsonData, err := json.MarshalIndent(saveData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal save data: %w", err)
	}

	savePath, err := getSaveFilePath(saveName)
	if err != nil {
		return "", fmt.Errorf("failed to get save path: %w", err)
	}

	saveDir := filepath.Dir(savePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	if err := os.WriteFile(savePath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write save file: %w", err)
	}

	return savePath, nil
}

func LoadGame(savePath string) (*domain.Player, *world.Level, error) {
	jsonData, err := os.ReadFile(savePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read save file: %w", err)
	}

	var saveData SaveData
	if err := json.Unmarshal(jsonData, &saveData); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal save data: %w", err)
	}

	if saveData.Version != 1 {
		return nil, nil, fmt.Errorf("unsupported save version: %d", saveData.Version)
	}

	player, err := deserializePlayer(&saveData.Player)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize player: %w", err)
	}

	level, err := deserializeWorld(&saveData.World, player)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize world: %w", err)
	}

	if err := deserializeEnemies(level, saveData.Enemies); err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize enemies: %w", err)
	}

	return player, level, nil
}

func serializePlayer(player *domain.Player) (PlayerData, error) {
	pd := PlayerData{
		Name:       player.Name,
		X:          player.X,
		Y:          player.Y,
		Gold:       player.Gold,
		Food:       player.Food,
		Life:       player.Life,
		ActiveDeck: player.ActiveDeck,
		Amulets:    make(map[int]int),
		Days:       player.Days,
	}

	if player.ActiveQuest != nil {
		q := player.ActiveQuest
		pd.ActiveQuest = &QuestData{
			Type:           int(q.Type),
			TargetCityName: q.TargetCityName,
			TargetEnemyID:  q.TargetEnemyID,
			EnemyName:      q.EnemyName,
			OriginCityName: q.OriginCityName,
			DaysRemaining:  q.DaysRemaining,
			RewardType:     int(q.RewardType),
			RewardAmulets:  q.RewardAmulets,
			AmuletColor:    int(q.AmuletColor),
			IsCompleted:    q.IsCompleted,
		}
		if q.RewardCard != nil {
			pd.ActiveQuest.RewardCardID = q.RewardCard.CardID()
		}
	}

	for colorMask, count := range player.Amulets {
		pd.Amulets[int(colorMask)] = count
	}

	pd.WorldMagics = make([]string, len(player.WorldMagics))
	for i, wm := range player.WorldMagics {
		pd.WorldMagics[i] = wm.Name
	}

	pd.CardCollection = make([]CardCollectionItem, 0, len(player.CardCollection))
	for card, item := range player.CardCollection {
		pd.CardCollection = append(pd.CardCollection, CardCollectionItem{
			CardID:     card.CardID(),
			TotalCount: item.Count,
			DeckCounts: item.DeckCounts,
		})
	}

	return pd, nil
}

func serializeWorld(level *world.Level) (WorldData, error) {
	w, h := level.Size()
	wd := WorldData{
		Width:  w,
		Height: h,
		Cities: make([]CityData, 0),
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tile := level.Tile(world.TilePoint{X: x, Y: y})
			if tile != nil && tile.IsCity {
				cityData := CityData{
					X:           x,
					Y:           y,
					Tier:        int(tile.City.Tier),
					Name:        tile.City.Name,
					AmuletColor: int(tile.City.AmuletColor),
					TerrainType: tile.TerrainType,
					QuestBanDays: tile.City.QuestBanDays,
					IsManaLinked: tile.City.IsManaLinked,
				}

				cityData.CardsForSale = make([]string, len(tile.City.CardsForSale))
				for i, card := range tile.City.CardsForSale {
					cityData.CardsForSale[i] = card.CardID()
				}

				if tile.City.AssignedWorldMagic != nil {
					cityData.AssignedWorldMagic = tile.City.AssignedWorldMagic.Name
				}

				wd.Cities = append(wd.Cities, cityData)
			}
		}
	}

	return wd, nil
}

func serializeEnemies(enemies []domain.Enemy) ([]EnemyData, error) {
	enemyData := make([]EnemyData, len(enemies))
	for i, enemy := range enemies {
		enemyData[i] = EnemyData{
			CharacterName: enemy.Character.Name,
			X:             enemy.X,
			Y:             enemy.Y,
			Engaged:       enemy.Engaged,
			MoveSpeed:     enemy.MoveSpeed,
		}
	}
	return enemyData, nil
}

func deserializePlayer(pd *PlayerData) (*domain.Player, error) {
	player, err := domain.NewPlayer(pd.Name, nil, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	player.X = pd.X
	player.Y = pd.Y
	player.Gold = pd.Gold
	player.Food = pd.Food
	player.Life = pd.Life
	player.ActiveDeck = pd.ActiveDeck
	player.Days = pd.Days

	if pd.ActiveQuest != nil {
		q := pd.ActiveQuest
		player.ActiveQuest = &domain.Quest{
			Type:           domain.QuestType(q.Type),
			TargetCityName: q.TargetCityName,
			TargetEnemyID:  q.TargetEnemyID,
			EnemyName:      q.EnemyName,
			OriginCityName: q.OriginCityName,
			DaysRemaining:  q.DaysRemaining,
			RewardType:     domain.RewardType(q.RewardType),
			RewardAmulets:  q.RewardAmulets,
			AmuletColor:    domain.ColorMask(q.AmuletColor),
			IsCompleted:    q.IsCompleted,
		}
		if q.RewardCardID != "" {
			player.ActiveQuest.RewardCard = findCardByID(q.RewardCardID)
		}
	}

	player.Amulets = make(map[domain.ColorMask]int)
	for colorMaskInt, count := range pd.Amulets {
		player.Amulets[domain.ColorMask(colorMaskInt)] = count
	}

	player.WorldMagics = make([]*domain.WorldMagic, 0, len(pd.WorldMagics))
	for _, name := range pd.WorldMagics {
		wm := findWorldMagicByName(name)
		if wm == nil {
			return nil, fmt.Errorf("world magic not found: %s", name)
		}
		player.WorldMagics = append(player.WorldMagics, wm)
	}

	player.CardCollection = domain.NewCardCollection()
	for _, item := range pd.CardCollection {
		card := findCardByID(item.CardID)
		if card == nil {
			return nil, fmt.Errorf("card not found: %s", item.CardID)
		}

		player.CardCollection[card] = &domain.CollectionItem{
			Card:       card,
			Count:      item.TotalCount,
			DeckCounts: item.DeckCounts,
		}
	}

	return player, nil
}

func deserializeWorld(wd *WorldData, player *domain.Player) (*world.Level, error) {
	level, err := world.NewLevel(player)
	if err != nil {
		return nil, fmt.Errorf("failed to create level: %w", err)
	}

	w, h := level.Size()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tile := level.Tile(world.TilePoint{X: x, Y: y})
			if tile != nil {
				tile.IsCity = false
			}
		}
	}

	for _, cityData := range wd.Cities {
		tile := level.Tile(world.TilePoint{X: cityData.X, Y: cityData.Y})
		if tile == nil {
			return nil, fmt.Errorf("invalid city location: (%d, %d)", cityData.X, cityData.Y)
		}

		city := domain.City{
			Tier:        domain.CityTier(cityData.Tier),
			Name:        cityData.Name,
			X:           cityData.X,
			Y:           cityData.Y,
			AmuletColor: domain.ColorMask(cityData.AmuletColor),
			QuestBanDays: cityData.QuestBanDays,
			IsManaLinked: cityData.IsManaLinked,
		}

		city.CardsForSale = make([]*domain.Card, len(cityData.CardsForSale))
		for i, cardID := range cityData.CardsForSale {
			card := findCardByID(cardID)
			if card == nil {
				return nil, fmt.Errorf("card not found for city: %s", cardID)
			}
			city.CardsForSale[i] = card
		}

		if cityData.AssignedWorldMagic != "" {
			wm := findWorldMagicByName(cityData.AssignedWorldMagic)
			if wm == nil {
				return nil, fmt.Errorf("world magic not found: %s", cityData.AssignedWorldMagic)
			}
			city.AssignedWorldMagic = wm
		}

		tile.IsCity = true
		tile.City = city
	}

	return level, nil
}

func deserializeEnemies(level *world.Level, enemiesData []EnemyData) error {
	level.ClearEnemies()

	enemies := make([]domain.Enemy, len(enemiesData))
	for i, ed := range enemiesData {
		enemy, err := domain.NewEnemy(ed.CharacterName)
		if err != nil {
			return fmt.Errorf("failed to create enemy %s: %w", ed.CharacterName, err)
		}

		enemy.X = ed.X
		enemy.Y = ed.Y
		enemy.Engaged = ed.Engaged
		enemy.MoveSpeed = ed.MoveSpeed

		enemies[i] = enemy
	}

	level.SetEnemies(enemies)
	return nil
}

func findCardByID(cardID string) *domain.Card {
	for _, card := range domain.CARDS {
		if card.CardID() == cardID {
			return card
		}
	}
	return nil
}

func findWorldMagicByName(name string) *domain.WorldMagic {
	for _, wm := range domain.AllWorldMagics {
		if wm.Name == name {
			return wm
		}
	}
	return nil
}

func getSaveFilePath(saveName string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	saveDir := filepath.Join(homeDir, ".s30", "saves")

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("%s_%s.json", saveName, timestamp)

	return filepath.Join(saveDir, filename), nil
}
