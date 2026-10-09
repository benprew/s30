package domain

import (
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"slices"
	"strings"

	_ "github.com/benprew/mage-go/cards"
	mage "github.com/benprew/mage-go/pkg/mage"
)

const ArenaEntryCost = 150
const ArenaRounds = 7

// ArenaRun keeps the temporary card pool separate from campaign resources.
type ArenaRun struct {
	Campaign *Player
	Player   *Player
	Round    int
	Finished bool
	Champion bool
	rng      *rand.Rand
}

// NewArenaRun charges the entry fee and creates the first round's card pool.
func NewArenaRun(campaign *Player, rng *rand.Rand) (*ArenaRun, error) {
	if campaign.Gold < ArenaEntryCost {
		return nil, fmt.Errorf("arena entry requires %d gold", ArenaEntryCost)
	}
	character := campaign.Character
	character.CardCollection = NewCardCollection()
	player := &Player{Character: character, Name: campaign.Name, MinDeckSize: 15}
	run := &ArenaRun{Campaign: campaign, Player: player, Round: 1, rng: rng}
	for _, name := range arenaLandNames {
		player.CardCollection.AddCard(FindCardByName(name), 20)
	}
	run.openPack()
	campaign.Gold -= ArenaEntryCost
	return run, nil
}

var arenaLandNames = []string{"Plains", "Island", "Swamp", "Mountain", "Forest"}
var arenaAntePattern = regexp.MustCompile(`(?i)\bante\b`)

func arenaAnteCard(card *Card) bool {
	return arenaAntePattern.MatchString(card.Text)
}

// ArenaBooster selects 10 common, 3 uncommon, and 1 rare card from all sets.
func ArenaBooster(rng *rand.Rand) []*Card {
	pools := map[string][]*Card{}
	seen := map[string]bool{}
	for _, card := range CARDS {
		if seen[card.CardName] || IsBasicLand(card) || arenaAnteCard(card) || !mage.CardRegistered(card.CardName) {
			continue
		}
		seen[card.CardName] = true
		pools[card.Rarity] = append(pools[card.Rarity], card)
	}
	var pack []*Card
	for _, slot := range []struct {
		rarity string
		count  int
	}{{"common", 10}, {"uncommon", 3}, {"rare", 1}} {
		pool := pools[slot.rarity]
		for _, index := range rng.Perm(len(pool))[:min(slot.count, len(pool))] {
			pack = append(pack, pool[index])
		}
	}
	return pack
}

func (a *ArenaRun) openPack() {
	for _, card := range ArenaBooster(a.rng) {
		a.Player.CardCollection.AddCard(card, 1)
	}
}

// NewOpponent uses a private copy of a rogue and a pool of one pack per round.
func (a *ArenaRun) NewOpponent() (*Enemy, error) {
	var candidates []*Character
	for _, rogue := range Rogues {
		if rogue.Level == a.Round {
			candidates = append(candidates, rogue)
		}
	}
	slices.SortFunc(candidates, func(a, b *Character) int { return strings.Compare(a.Name, b.Name) })
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no arena opponents for round %d", a.Round)
	}
	character := *candidates[a.rng.Intn(len(candidates))]
	var pool []*Card
	for range a.Round {
		pool = append(pool, ArenaBooster(a.rng)...)
	}
	character.CardCollection = arenaOpponentCollection(pool, a.Player.MinDeckSize)
	character.PrimaryColor, character.ColorIdentity = analyzeColors(character.CardCollection)
	enemy := NewEnemyFromCharacter(&character)
	return &enemy, nil
}

func arenaOpponentCollection(pool []*Card, minimum int) CardCollection {
	colors := []string{"W", "U", "B", "R", "G"}
	weights := map[string]int{}
	for _, card := range pool {
		if card.CardType == CardTypeLand {
			continue
		}
		weight := arenaCardScore(card)
		for _, c := range card.ColorIdentity {
			weights[c] += weight
		}
	}
	slices.SortStableFunc(colors, func(a, b string) int { return weights[b] - weights[a] })
	chosen := colors[:2]
	collection := NewCardCollection()
	var spells, openedLands []*Card
	mana := map[string]int{}
	for _, card := range pool {
		if !arenaColorsFit(card, chosen) {
			continue
		}
		if card.CardType == CardTypeLand {
			openedLands = append(openedLands, card)
		} else {
			spells = append(spells, card)
		}
	}
	rank := func(a, b *Card) int {
		if score := arenaCardScore(b) - arenaCardScore(a); score != 0 {
			return score
		}
		return strings.Compare(a.CardName, b.CardName)
	}
	slices.SortStableFunc(spells, rank)
	slices.SortStableFunc(openedLands, rank)
	selectedSpells := spells[:0]
	for count := min(len(spells), minimum-1); count > 0; count-- {
		if count+arenaLandCount(minimum, spells[:count]) <= minimum {
			selectedSpells = spells[:count]
			break
		}
	}
	for _, card := range selectedSpells {
		collection.AddCardToDeck(card, 0, 1)
		for _, c := range card.ColorIdentity {
			mana[c]++
		}
	}
	for _, card := range openedLands[:min(len(openedLands), minimum-collection.NumCards())] {
		collection.AddCardToDeck(card, 0, 1)
	}
	lands := minimum - collection.NumCards()
	manaTotal := max(1, mana[chosen[0]]+mana[chosen[1]])
	for i := range lands {
		c := chosen[0]
		if (i+1)*mana[chosen[1]]/manaTotal > i*mana[chosen[1]]/manaTotal {
			c = chosen[1]
		}
		for j, symbol := range []string{"W", "U", "B", "R", "G"} {
			if c == symbol {
				land := FindCardByName(arenaLandNames[j])
				if collection.GetDeckCount(land, 0) >= 20 {
					for _, name := range arenaLandNames {
						candidate := FindCardByName(name)
						if collection.GetDeckCount(candidate, 0) < 20 {
							land = candidate
							break
						}
					}
				}
				collection.AddCardToDeck(land, 0, 1)
			}
		}
	}
	return collection
}

func arenaCardScore(card *Card) int {
	tier, ok := CardTierForName(card.CardName)
	if !ok {
		tier = TierF
	}
	score := int(TierF-tier) + 1
	if card.CardType == CardTypeCreature {
		score++
	}
	return score
}

var arenaDrawPattern = regexp.MustCompile(`(?i)\bdraws? (?:a|one|two|three|\d+) cards?\b`)

func arenaCheapManaSupport(card *Card) bool {
	if card.CardType == CardTypeLand || card.ManaValue() > 2 {
		return false
	}
	if len(card.ManaProduction) > 0 {
		return true
	}
	return (card.CardType == CardTypeInstant || card.CardType == CardTypeSorcery) && arenaDrawPattern.MatchString(card.Text)
}

func arenaLandCount(size int, spells []*Card) int {
	if len(spells) == 0 {
		return size
	}
	manaTotal, highestMana, support := 0, 0, 0
	for _, card := range spells {
		value := card.ManaValue()
		manaTotal += value
		highestMana = max(highestMana, value)
		if arenaCheapManaSupport(card) {
			support++
		}
	}
	averageMana := float64(manaTotal) / float64(len(spells))
	ratio := 0.3265 + 0.0317*averageMana
	aggressive := averageMana <= 1.2 && highestMana <= 2
	if aggressive {
		ratio = min(ratio, 0.33)
	}
	lands := float64(size)*ratio - 0.28*float64(support)
	if aggressive {
		lands = max(lands, float64(size)*0.30)
	}
	return min(size, max(1, int(math.Round(lands))))
}

func arenaColorsFit(card *Card, colors []string) bool {
	for _, c := range card.ColorIdentity {
		if !slices.Contains(colors, c) {
			return false
		}
	}
	return true
}

// Win grants the round reward and opens the next pack unless this is the final win.
func (a *ArenaRun) Win() DuelReward {
	if a.Finished {
		return DuelReward{}
	}
	reward := arenaReward(a.Round, a.rng)
	a.Campaign.Gold += reward.Gold
	for _, card := range reward.Cards {
		a.Campaign.CardCollection.AddCard(card, 1)
	}
	for _, amulet := range reward.Amulets {
		a.Campaign.AddAmulet(amulet)
	}
	if a.Round == ArenaRounds {
		a.Champion = true
		a.Finish()
	} else {
		a.Round++
		a.Player.MinDeckSize = 10 + 5*a.Round
		a.openPack()
	}
	return reward
}

// Finish ends the run without transferring its temporary cards.
func (a *ArenaRun) Finish() {
	a.Finished = true
}

func arenaReward(round int, rng *rand.Rand) DuelReward {
	gold := []int{50, 50, 25, 25, 50, 100, 100}
	counts := []int{1, 1, 2, 2, 2, 2, 2}
	amulets := []int{0, 1, 1, 1, 1, 1, 2}
	tiers := [][]CardTier{{TierD, TierF}, {TierD, TierF}, {TierD, TierF}, {TierC, TierD}, {TierB, TierC}, {TierA, TierB}, {TierA, TierB}}
	reward := DuelReward{Gold: gold[round-1]}
	pool := CardsInTiers(tiers[round-1]...)
	for _, index := range rng.Perm(len(pool))[:counts[round-1]] {
		reward.Cards = append(reward.Cards, pool[index])
	}
	if round == ArenaRounds {
		bonus := pool
		if rng.Float64() < 0.03 {
			bonus = CardsInTiers(TierS)
		}
		reward.Cards = append(reward.Cards, bonus[rng.Intn(len(bonus))])
	}
	colors := GetAllAmuletColors()
	for range amulets[round-1] {
		reward.Amulets = append(reward.Amulets, NewAmulet(colors[rng.Intn(len(colors))]))
	}
	return reward
}
