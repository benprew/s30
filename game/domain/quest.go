package domain

type QuestType int

const (
	QuestTypeDelivery QuestType = iota
	QuestTypeDefeatEnemy
	// QuestTypeActionTracker accumulates progress from in-duel actions across
	// one or more duels until Target is reached.
	QuestTypeActionTracker
	// QuestTypeDeckConstraint completes on a duel win while the submitted deck
	// (or the player's play) satisfies a constraint.
	QuestTypeDeckConstraint
)

// QuestReward is the unified reward any quest can grant. Every quest type
// (delivery, defeat-enemy, action-tracker, deck-constraint) uses this same
// shape; legacy delivery/defeat quests simply grant one of a single kind
// (e.g. one mana link), while deck-changing quests may grant a bundle.
type QuestReward struct {
	Gold        int
	Cards       int       // number of cards granted (drawn from CardColor's pool)
	CardColor   ColorMask // color bundle for Cards (ColorColorless => any)
	Amulets     int
	AmuletColor ColorMask
	ManaLinks   int // each grants a permanent +1 life
}

// IsEmpty reports whether the reward grants nothing.
func (r QuestReward) IsEmpty() bool {
	return r.Gold == 0 && r.Cards == 0 && r.Amulets == 0 && r.ManaLinks == 0
}

// QuestMetric identifies which in-duel action an action-tracker quest counts.
type QuestMetric int

const (
	MetricNone QuestMetric = iota
	// MetricCastColor counts spells cast whose color matches Color.
	MetricCastColor
	// MetricPlayLands counts lands played.
	MetricPlayLands
	// MetricAttackCreatures counts creatures declared as attackers.
	MetricAttackCreatures
	// MetricDestroyEnemyCreatures counts the opponent's creatures that died.
	MetricDestroyEnemyCreatures
	// MetricCastType counts spells cast of CardTypes.
	MetricCastType
	// MetricDirectDamage counts non-combat damage dealt to the opponent.
	MetricDirectDamage
)

// QuestConstraint identifies the deck/play constraint a deck-constraint quest
// validates. Checked at duel start (deck constraints) or at duel end
// (ConstraintNoAttacking).
type QuestConstraint int

const (
	ConstraintNone QuestConstraint = iota
	// ConstraintMonoColor requires all non-land cards to share a single color.
	ConstraintMonoColor
	// ConstraintFatDeck requires the deck size to be > ConstraintN. A bloated
	// deck is harder to win with (lower odds of drawing your best cards).
	ConstraintFatDeck
	// ConstraintLowCurve requires every creature's mana value to be <= ConstraintN.
	ConstraintLowCurve
	// ConstraintColorLight requires no card of Color in the deck.
	ConstraintColorLight
	// ConstraintNoAttacking requires winning without declaring an attacker.
	ConstraintNoAttacking
)

type Quest struct {
	Type          QuestType
	TargetCity    *City  // delivery only: the city the message must reach
	EnemyName     string // defeat-enemy only: display name of the target enemy
	DaysRemaining int
	IsCompleted   bool

	ID          string // template id (shared by every quest generated from it)
	Title       string
	Description string

	// Template data. DeadlineDays seeds DaysRemaining and RewardTier drives the
	// Reward when GenerateQuest instantiates a concrete quest from a template.
	DeadlineDays int
	RewardTier   QuestRewardTier

	// Objective. Color is shared by every color-based objective: cast-color
	// (action), mono-color and color-light (constraint).
	Metric      QuestMetric     // action-tracker objective
	Constraint  QuestConstraint // deck-constraint objective
	Color       ColorMask
	CardTypes   []CardType // for MetricCastType (e.g. instant + sorcery)
	Target      int        // action-tracker count target
	Progress    int        // accumulated action-tracker progress
	ConstraintN int        // for ConstraintFatDeck / ConstraintLowCurve

	Reward QuestReward
}

// MaxActiveQuests is the number of quests a player may hold at once. Every
// quest type (delivery, defeat-enemy, action-tracker, deck-constraint) shares
// these slots.
const MaxActiveQuests = 3

// IsFulfilled reports whether the quest's objective has been met and it is
// ready to be redeemed. Action trackers complete when progress reaches the
// target; every other type completes when IsCompleted is set — delivery on
// reaching its target city, defeat-enemy on the kill, deck-constraint on a
// qualifying win.
func (q *Quest) IsFulfilled() bool {
	if q.Type == QuestTypeActionTracker {
		return q.Progress >= q.Target
	}
	return q.IsCompleted
}

// AddProgress increases an action-tracker quest's progress, clamped to Target.
func (q *Quest) AddProgress(n int) {
	if q.Type != QuestTypeActionTracker || n <= 0 {
		return
	}
	q.Progress += n
	if q.Progress > q.Target {
		q.Progress = q.Target
	}
}
