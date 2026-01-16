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
	Type           QuestType
	TargetCityName string
	TargetEnemyID  string // or Name
	EnemyName      string // Display name
	OriginCityName string
	DaysRemaining  int
	RewardType     RewardType
	RewardCard     *Card
	RewardAmulets  int
	AmuletColor    ColorMask
	IsCompleted    bool
}
