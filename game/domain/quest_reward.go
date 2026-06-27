package domain

import (
	"fmt"
	"math/rand"
	"strings"
)

// Reward scaling for deck-changing quests. Amounts grow with player
// progression (see world.progressionEnemyMaxLevel, fed in as `progression`).
const (
	rewardGoldStandardBase   = 60
	rewardGoldStandardScale  = 25
	rewardGoldChallengeBase  = 100
	rewardGoldChallengeScale = 40
	rewardGoldThemedBase     = 150
	rewardGoldThemedScale    = 60
)

// DeckQuestReward records what a redeemed quest paid out, for the reward screen.
type DeckQuestReward struct {
	Quest  *Quest
	Reward QuestReward
	Cards  []*Card
}

// GrantQuestReward applies a reward to the player and returns any cards added
// (so the reward screen can display them). Every quest type funnels its payout
// through here.
func GrantQuestReward(p *Player, r QuestReward) []*Card {
	p.Gold += r.Gold
	p.Life += r.ManaLinks
	for range r.Amulets {
		p.AddAmulet(NewAmulet(r.AmuletColor))
	}
	var cards []*Card
	if r.Cards > 0 {
		color := r.CardColor
		if color == ColorColorless {
			color = ColorAny
		}
		cards = RandomPowerfulCardsForColor(color, r.Cards)
		for _, c := range cards {
			p.CardCollection.AddCard(c, 1)
		}
	}
	return cards
}

// Description renders a human-readable summary like "120 gold and a card".
func (r QuestReward) Description() string {
	var parts []string
	if r.Gold > 0 {
		parts = append(parts, fmt.Sprintf("%d gold", r.Gold))
	}
	parts = appendCount(parts, r.Cards, "a card", "%d cards")
	if r.Amulets == 1 {
		parts = append(parts, fmt.Sprintf("a %s amulet", ColorMaskToString(r.AmuletColor)))
	} else if r.Amulets > 1 {
		parts = append(parts, fmt.Sprintf("%d %s amulets", r.Amulets, ColorMaskToString(r.AmuletColor)))
	}
	parts = appendCount(parts, r.ManaLinks, "a mana link", "%d mana links")
	if len(parts) == 0 {
		return "a reward"
	}
	return strings.Join(parts, " and ")
}

func appendCount(parts []string, n int, one, many string) []string {
	switch {
	case n == 1:
		return append(parts, one)
	case n > 1:
		return append(parts, fmt.Sprintf(many, n))
	}
	return parts
}

// RandomSingleReward builds a reward of one randomly chosen kind, the payout for
// legacy delivery/defeat quests ("1 of a single type"). amuletColor is the city
// color used when an amulet is rolled.
func RandomSingleReward(progression int, amuletColor ColorMask) QuestReward {
	if progression < 0 {
		progression = 0
	}
	switch rand.Intn(4) {
	case 0:
		return QuestReward{ManaLinks: 1}
	case 1:
		return QuestReward{Amulets: 1, AmuletColor: amuletColor}
	case 2:
		return QuestReward{Cards: 1, CardColor: ColorAny}
	default:
		return QuestReward{Gold: rewardGoldStandardBase + progression*rewardGoldStandardScale}
	}
}

// GenerateQuest builds a concrete deck-changing Quest from this template,
// scaling reward amounts to the given progression level (0 = earliest game).
func (t *Quest) GenerateQuest(progression int) *Quest {
	if progression < 0 {
		progression = 0
	}
	q := *t
	q.DaysRemaining = t.DeadlineDays
	q.Reward = t.buildReward(progression)
	q.Progress = 0
	q.IsCompleted = false
	return &q
}

func (d *Quest) buildReward(progression int) QuestReward {
	switch d.RewardTier {
	case TierThemed:
		return QuestReward{
			Gold:      rewardGoldThemedBase + progression*rewardGoldThemedScale,
			Cards:     2,
			CardColor: d.rewardColor(),
		}
	case TierChallenge:
		return QuestReward{
			Gold:      rewardGoldChallengeBase + progression*rewardGoldChallengeScale,
			Cards:     1,
			CardColor: d.rewardColor(),
		}
	default: // TierStandard
		return QuestReward{Gold: rewardGoldStandardBase + progression*rewardGoldStandardScale}
	}
}

// rewardColor picks the color a quest's card bundle should favor. Falls back to
// ColorAny when the quest has no color theme.
func (d *Quest) rewardColor() ColorMask {
	if d.Color != ColorColorless {
		return d.Color
	}
	return ColorAny
}
