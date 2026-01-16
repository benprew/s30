package save

import "time"

type SaveData struct {
	Version int       `json:"version"`
	SavedAt time.Time `json:"saved_at"`
	Player  PlayerData `json:"player"`
	World   WorldData  `json:"world"`
	Enemies []EnemyData `json:"enemies"`
}

type PlayerData struct {
	Name           string              `json:"name"`
	X              int                 `json:"x"`
	Y              int                 `json:"y"`
	Gold           int                 `json:"gold"`
	Food           int                 `json:"food"`
	Life           int                 `json:"life"`
	ActiveDeck     int                 `json:"active_deck"`
	Amulets        map[int]int         `json:"amulets"`
	WorldMagics    []string            `json:"world_magics"`
	CardCollection []CardCollectionItem `json:"card_collection"`
	ActiveQuest    *QuestData          `json:"active_quest,omitempty"`
	Days           int                 `json:"days"`
}

type QuestData struct {
	Type           int    `json:"type"`
	TargetCityName string `json:"target_city_name"`
	TargetEnemyID  string `json:"target_enemy_id"`
	EnemyName      string `json:"enemy_name"`
	OriginCityName string `json:"origin_city_name"`
	DaysRemaining  int    `json:"days_remaining"`
	RewardType     int    `json:"reward_type"`
	RewardCardID   string `json:"reward_card_id,omitempty"`
	RewardAmulets  int    `json:"reward_amulets"`
	AmuletColor    int    `json:"amulet_color"`
	IsCompleted    bool   `json:"is_completed"`
}

type CardCollectionItem struct {
	CardID     string `json:"card_id"`
	TotalCount int    `json:"total_count"`
	DeckCounts []int  `json:"deck_counts"`
}

type WorldData struct {
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Cities []CityData `json:"cities"`
}

type CityData struct {
	X                  int      `json:"x"`
	Y                  int      `json:"y"`
	Tier               int      `json:"tier"`
	Name               string   `json:"name"`
	AmuletColor        int      `json:"amulet_color"`
	CardsForSale       []string `json:"cards_for_sale"`
	AssignedWorldMagic string   `json:"assigned_world_magic,omitempty"`
	TerrainType        int      `json:"terrain_type"`
	QuestBanDays       int      `json:"quest_ban_days"`
	IsManaLinked       bool     `json:"is_mana_linked"`
}

type EnemyData struct {
	CharacterName string `json:"character_name"`
	X             int    `json:"x"`
	Y             int    `json:"y"`
	Engaged       bool   `json:"engaged"`
	MoveSpeed     int    `json:"move_speed"`
}
