package domain

type QuestType int

const (
	QuestTypeDelivery QuestType = iota
	QuestTypeDefeatEnemy
)

type RewardType int

const (
	RewardManaLink RewardType = iota
	RewardAmulet
	RewardCard
)

type Quest struct {
	Type          QuestType
	TargetCity    *City
	TargetEnemyID string // or Name
	EnemyName     string // Display name
	OriginCity    *City
	DaysRemaining int
	RewardType    RewardType
	RewardCard    *Card
	RewardAmulets int
	AmuletColor   ColorMask
	IsCompleted   bool
}
