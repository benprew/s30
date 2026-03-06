package core

import (
	"testing"

	"github.com/benprew/s30/game/domain"
	"github.com/benprew/s30/mtg/effects"
)

func addCreatureWithMenace(player *Player) *Card {
	domainCard := domain.FindCardByName("Llanowar Elves")
	if domainCard == nil {
		return nil
	}
	card := NewCardFromDomain(domainCard, EntityID(200+len(player.Battlefield)), player)
	card.Active = true
	card.Tapped = false
	card.CurrentZone = ZoneBattlefield
	card.ParsedAbilities = append(card.ParsedAbilities, domain.ParsedAbility{
		Type:     "keyword",
		Keywords: []string{"Menace"},
		Effect:   &domain.ParsedEffect{Keywords: []string{"Menace"}},
	})
	player.Battlefield = append(player.Battlefield, card)
	return card
}

func TestMenace_SingleBlockerRemovedDealsDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureWithMenace(player)
	if attacker == nil {
		t.Skip("could not create menace creature")
	}
	attacker.Power = 3
	attacker.Toughness = 3

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.DeclareBlocker(blocker, attacker)
	game.ValidateMenaceBlocks()

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife-attacker.Power {
		t.Errorf("menace attacker with single blocker removed should deal %d damage to player, opponent life: got %d want %d",
			attacker.Power, opponent.LifeTotal, initialLife-attacker.Power)
	}

	if blocker.DamageTaken != 0 {
		t.Errorf("removed blocker should take no damage, got %d", blocker.DamageTaken)
	}
}

func TestMenace_TwoBlockersStopDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureWithMenace(player)
	if attacker == nil {
		t.Skip("could not create menace creature")
	}
	attacker.Power = 3
	attacker.Toughness = 3

	blocker1 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	blocker2 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker1 == nil || blocker2 == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker1.Power = 1
	blocker1.Toughness = 1
	blocker2.Power = 1
	blocker2.Toughness = 1

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.DeclareBlocker(blocker1, attacker)
	game.DeclareBlocker(blocker2, attacker)
	game.ValidateMenaceBlocks()

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife {
		t.Errorf("menace attacker blocked by two creatures should not deal damage to player, opponent life: got %d want %d",
			opponent.LifeTotal, initialLife)
	}

	if blocker1.DamageTaken == 0 && blocker2.DamageTaken == 0 {
		t.Errorf("at least one blocker should take damage from attacker")
	}
}

func TestMenace_UnblockedDealsDamageToPlayer(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureWithMenace(player)
	if attacker == nil {
		t.Skip("could not create menace creature")
	}
	attacker.Power = 4

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.ValidateMenaceBlocks()

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife-attacker.Power {
		t.Errorf("unblocked menace attacker should deal %d damage, opponent life: got %d want %d",
			attacker.Power, opponent.LifeTotal, initialLife-attacker.Power)
	}
}

func TestMenace_ThreeBlockersAllowed(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureWithMenace(player)
	if attacker == nil {
		t.Skip("could not create menace creature")
	}

	blocker1 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	blocker2 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	blocker3 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker1 == nil || blocker2 == nil || blocker3 == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.Attackers = append(game.Attackers, attacker)
	game.DeclareBlocker(blocker1, attacker)
	game.DeclareBlocker(blocker2, attacker)
	game.DeclareBlocker(blocker3, attacker)
	game.ValidateMenaceBlocks()

	blockers := game.BlockerMap[attacker]
	if len(blockers) != 3 {
		t.Errorf("menace creature blocked by three should keep all blockers, got %d", len(blockers))
	}
}

func TestMenace_NonMenaceCreatureSingleBlockerKept(t *testing.T) {
	game, player, opponent := setupCombatTest()

	attacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if attacker == nil {
		t.Skip("Llanowar Elves not found")
	}
	attacker.Power = 2
	attacker.Toughness = 2

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.ActivePlayer = 0
	game.DeclareAttacker(attacker)
	game.Attackers = append(game.Attackers, attacker)
	game.DeclareBlocker(blocker, attacker)
	game.ValidateMenaceBlocks()

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	if opponent.LifeTotal != initialLife {
		t.Errorf("blocked non-menace attacker should not deal damage to player, opponent life: got %d want %d",
			opponent.LifeTotal, initialLife)
	}
}

func TestMenace_MixedAttackersMenaceAndNormal(t *testing.T) {
	game, player, opponent := setupCombatTest()

	menaceAttacker := addCreatureWithMenace(player)
	normalAttacker := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if menaceAttacker == nil || normalAttacker == nil {
		t.Skip("could not create creatures")
	}
	menaceAttacker.Power = 3
	menaceAttacker.Toughness = 3
	normalAttacker.Power = 2
	normalAttacker.Toughness = 2

	blocker1 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	blocker2 := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker1 == nil || blocker2 == nil {
		t.Skip("Llanowar Elves not found")
	}
	blocker1.Power = 1
	blocker1.Toughness = 4
	blocker2.Power = 1
	blocker2.Toughness = 4

	game.ActivePlayer = 0
	game.DeclareAttacker(menaceAttacker)
	game.Attackers = append(game.Attackers, menaceAttacker)
	game.DeclareAttacker(normalAttacker)
	game.Attackers = append(game.Attackers, normalAttacker)

	game.DeclareBlocker(blocker1, menaceAttacker)
	game.DeclareBlocker(blocker2, normalAttacker)
	game.ValidateMenaceBlocks()

	initialLife := opponent.LifeTotal
	game.ResolveCombatDamage()

	menaceBlockers := game.BlockerMap[menaceAttacker]
	if len(menaceBlockers) != 0 {
		t.Errorf("menace attacker's single blocker should be removed, got %d blockers", len(menaceBlockers))
	}

	normalBlockers := game.BlockerMap[normalAttacker]
	if len(normalBlockers) != 1 {
		t.Errorf("normal attacker's single blocker should be kept, got %d blockers", len(normalBlockers))
	}

	expectedLife := initialLife - menaceAttacker.Power
	if opponent.LifeTotal != expectedLife {
		t.Errorf("only menace attacker should deal player damage, opponent life: got %d want %d",
			opponent.LifeTotal, expectedLife)
	}
}

func addEnchantmentWithLordKeyword(player *Player, keyword string) *Card {
	card := &Card{
		Card: domain.Card{
			CardName: "Test Enchantment",
			CardType: domain.CardTypeEnchantment,
			ParsedAbilities: []domain.ParsedAbility{
				{
					Type: "Static",
					Effect: &domain.ParsedEffect{
						GrantedKeyword: keyword,
					},
				},
			},
		},
		ID:          EntityID(300 + len(player.Battlefield)),
		Owner:       player,
		CurrentZone: ZoneBattlefield,
	}
	player.Battlefield = append(player.Battlefield, card)
	return card
}

func TestLordEffect_EnchantmentGrantsMenace(t *testing.T) {
	_, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	if creature.HasKeyword(effects.KeywordMenace) {
		t.Error("creature should not have menace before enchantment")
	}

	addEnchantmentWithLordKeyword(player, string(effects.KeywordMenace))

	if !creature.HasKeyword(effects.KeywordMenace) {
		t.Error("creature should have menace from lord enchantment")
	}
}

func TestLordEffect_EnchantmentDoesNotAffectOpponent(t *testing.T) {
	_, player, opponent := setupCombatTest()

	addEnchantmentWithLordKeyword(player, string(effects.KeywordMenace))

	oppCreature := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if oppCreature == nil {
		t.Skip("Llanowar Elves not found")
	}

	if oppCreature.HasKeyword(effects.KeywordMenace) {
		t.Error("opponent creature should not have menace from player's enchantment")
	}
}

func TestLordEffect_MenaceFromEnchantmentBlockValidation(t *testing.T) {
	game, player, opponent := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}
	creature.Power = 3
	creature.Toughness = 3

	addEnchantmentWithLordKeyword(player, string(effects.KeywordMenace))

	blocker := addCreatureToBattlefield(opponent, "Llanowar Elves", true, false)
	if blocker == nil {
		t.Skip("Llanowar Elves not found")
	}

	game.ActivePlayer = 0
	game.DeclareAttacker(creature)
	game.Attackers = append(game.Attackers, creature)
	game.DeclareBlocker(blocker, creature)
	game.ValidateMenaceBlocks()

	blockers := game.BlockerMap[creature]
	if len(blockers) != 0 {
		t.Errorf("menace creature (from enchantment) with single blocker should have blocker removed, got %d", len(blockers))
	}
}

func TestLordEffect_RemovedEnchantmentRemovesMenace(t *testing.T) {
	_, player, _ := setupCombatTest()

	creature := addCreatureToBattlefield(player, "Llanowar Elves", true, false)
	if creature == nil {
		t.Skip("Llanowar Elves not found")
	}

	enchantment := addEnchantmentWithLordKeyword(player, string(effects.KeywordMenace))

	if !creature.HasKeyword(effects.KeywordMenace) {
		t.Error("creature should have menace from enchantment")
	}

	for i, c := range player.Battlefield {
		if c == enchantment {
			player.Battlefield = append(player.Battlefield[:i], player.Battlefield[i+1:]...)
			break
		}
	}

	if creature.HasKeyword(effects.KeywordMenace) {
		t.Error("creature should not have menace after enchantment removed")
	}
}
