package parser

import (
	"testing"

	"github.com/benprew/s30/mtg/effects"
)

func TestParseKeywords(t *testing.T) {
	tests := []struct {
		name     string
		cardName string
		text     string
		keywords []Keyword
	}{
		{"Flying", "Serra Angel", "Flying", []Keyword{KeywordFlying}},
		{"Vigilance", "Serra Angel", "Vigilance", []Keyword{KeywordVigilance}},
		{"First Strike", "White Knight", "First strike", []Keyword{KeywordFirstStrike}},
		{"Trample", "Craw Wurm", "Trample", []Keyword{KeywordTrample}},
		{"Haste", "Ball Lightning", "Haste", []Keyword{KeywordHaste}},
		{"Banding", "Benalish Hero", "Banding", []Keyword{KeywordBanding}},
		{"Fear", "Black Knight", "Fear", []Keyword{KeywordFear}},
		{"Protection", "Black Knight", "Protection from white", []Keyword{KeywordProtection}},
		{"Swampwalk", "Bog Wraith", "Swampwalk", []Keyword{KeywordLandwalk}},
		{"Islandwalk", "Lord of Atlantis", "Islandwalk", []Keyword{KeywordLandwalk}},
		{"Forestwalk", "Shanodin Dryads", "Forestwalk", []Keyword{KeywordLandwalk}},
		{"Mountainwalk", "Goblin King", "Mountainwalk", []Keyword{KeywordLandwalk}},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			if len(ability.Keywords) != len(tt.keywords) {
				t.Errorf("expected %d keywords, got %d", len(tt.keywords), len(ability.Keywords))
				return
			}
			for i, kw := range tt.keywords {
				if ability.Keywords[i] != kw {
					t.Errorf("expected keyword %v, got %v", kw, ability.Keywords[i])
				}
			}
		})
	}
}

func TestParseDamageSpells(t *testing.T) {
	tests := []struct {
		name       string
		cardName   string
		text       string
		amount     int
		targetType TargetType
	}{
		{"Lightning Bolt", "Lightning Bolt", "Lightning Bolt deals 3 damage to any target", 3, TargetTypeAny},
		{"Shock", "Shock", "Shock deals 2 damage to any target", 2, TargetTypeAny},
		{"Incinerate", "Incinerate", "Incinerate deals 3 damage to any target", 3, TargetTypeAny},
		{"Chain Lightning", "Chain Lightning", "Chain Lightning deals 3 damage to any target", 3, TargetTypeAny},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			dmg, ok := ability.Effect.(*effects.DirectDamage)
			if !ok {
				t.Errorf("expected DirectDamage effect, got %T", ability.Effect)
				return
			}
			if dmg.Amount != tt.amount {
				t.Errorf("expected %d damage, got %d", tt.amount, dmg.Amount)
			}
			if ability.TargetSpec == nil || ability.TargetSpec.Type != tt.targetType {
				t.Errorf("expected target type %v", tt.targetType)
			}
		})
	}
}

func TestParseStatBoosts(t *testing.T) {
	tests := []struct {
		name      string
		cardName  string
		text      string
		power     int
		toughness int
	}{
		{"Giant Growth", "Giant Growth", "Target creature gets +3/+3 until end of turn", 3, 3},
		{"Holy Strength", "Holy Strength", "Enchanted creature gets +1/+2", 1, 2},
		{"Unholy Strength", "Unholy Strength", "Enchanted creature gets +2/+1", 2, 1},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			boost, ok := ability.Effect.(*effects.StatBoost)
			if !ok {
				t.Errorf("expected StatBoost effect, got %T", ability.Effect)
				return
			}
			if boost.PowerBoost != tt.power {
				t.Errorf("expected power boost %d, got %d", tt.power, boost.PowerBoost)
			}
			if boost.ToughnessBoost != tt.toughness {
				t.Errorf("expected toughness boost %d, got %d", tt.toughness, boost.ToughnessBoost)
			}
		})
	}
}

func TestParseManaAbilities(t *testing.T) {
	tests := []struct {
		name      string
		cardName  string
		text      string
		manaTypes []string
		anyColor  bool
	}{
		{"Llanowar Elves", "Llanowar Elves", "{T}: Add {G}", []string{"G"}, false},
		{"Birds of Paradise", "Birds of Paradise", "{T}: Add one mana of any color", []string{"W", "U", "B", "R", "G"}, true},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			mana, ok := ability.Effect.(*effects.ManaAbility)
			if !ok {
				t.Errorf("expected ManaAbility effect, got %T", ability.Effect)
				return
			}
			if mana.AnyColor != tt.anyColor {
				t.Errorf("expected anyColor %v, got %v", tt.anyColor, mana.AnyColor)
			}
			if len(mana.ManaTypes) != len(tt.manaTypes) {
				t.Errorf("expected %d mana types, got %d", len(tt.manaTypes), len(mana.ManaTypes))
			}
		})
	}
}

func TestParseLordEffects(t *testing.T) {
	tests := []struct {
		name      string
		cardName  string
		text      string
		subtype   string
		power     int
		toughness int
		keyword   effects.Keyword
		modifier  string
	}{
		{"Lord of Atlantis", "Lord of Atlantis", "Other Merfolk creatures you control get +1/+1 and have islandwalk", "Merfolk", 1, 1, effects.KeywordLandwalk, "islandwalk"},
		{"Goblin King", "Goblin King", "Other Goblin creatures you control get +1/+1 and have mountainwalk", "Goblin", 1, 1, effects.KeywordLandwalk, "mountainwalk"},
		{"Zombie Master", "Zombie Master", "Other Zombie creatures get +1/+1", "Zombie", 1, 1, "", ""},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			lord, ok := ability.Effect.(*effects.LordEffect)
			if !ok {
				t.Errorf("expected LordEffect, got %T", ability.Effect)
				return
			}
			if lord.Subtype != tt.subtype {
				t.Errorf("expected subtype %q, got %q", tt.subtype, lord.Subtype)
			}
			if lord.PowerBoost != tt.power {
				t.Errorf("expected power boost %d, got %d", tt.power, lord.PowerBoost)
			}
			if lord.ToughnessBoost != tt.toughness {
				t.Errorf("expected toughness boost %d, got %d", tt.toughness, lord.ToughnessBoost)
			}
			if tt.keyword != "" {
				if lord.GrantedKeyword == nil {
					t.Error("expected granted keyword, got nil")
				} else if *lord.GrantedKeyword != tt.keyword {
					t.Errorf("expected granted keyword %q, got %q", tt.keyword, *lord.GrantedKeyword)
				}
				if lord.GrantedModifier != tt.modifier {
					t.Errorf("expected granted modifier %q, got %q", tt.modifier, lord.GrantedModifier)
				}
			}
		})
	}
}

func TestParseActivatedDamage(t *testing.T) {
	tests := []struct {
		name     string
		cardName string
		text     string
		amount   int
		hasTap   bool
	}{
		{"Prodigal Sorcerer", "Prodigal Sorcerer", "{T}: Prodigal Sorcerer deals 1 damage to any target", 1, true},
		{"Tim", "Prodigal Sorcerer", "{T}: ~ deals 1 damage to any target", 1, true},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			if ability.Cost == nil || ability.Cost.Tap != tt.hasTap {
				t.Errorf("expected tap cost %v", tt.hasTap)
			}
			dmg, ok := ability.Effect.(*effects.DirectDamage)
			if !ok {
				t.Errorf("expected DirectDamage effect, got %T", ability.Effect)
				return
			}
			if dmg.Amount != tt.amount {
				t.Errorf("expected %d damage, got %d", tt.amount, dmg.Amount)
			}
		})
	}
}

func TestParseActivatedPump(t *testing.T) {
	tests := []struct {
		name      string
		cardName  string
		text      string
		power     int
		toughness int
		manaCost  string
	}{
		{"Shivan Dragon", "Shivan Dragon", "{R}: Shivan Dragon gets +1/+0 until end of turn", 1, 0, "{R}"},
		{"Frozen Shade", "Frozen Shade", "{B}: Frozen Shade gets +1/+1 until end of turn", 1, 1, "{B}"},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			boost, ok := ability.Effect.(*effects.StatBoost)
			if !ok {
				t.Errorf("expected StatBoost effect, got %T", ability.Effect)
				return
			}
			if boost.PowerBoost != tt.power {
				t.Errorf("expected power boost %d, got %d", tt.power, boost.PowerBoost)
			}
			if boost.ToughnessBoost != tt.toughness {
				t.Errorf("expected toughness boost %d, got %d", tt.toughness, boost.ToughnessBoost)
			}
		})
	}
}

func TestParseTriggeredAbilities(t *testing.T) {
	tests := []struct {
		name        string
		cardName    string
		text        string
		triggerType TriggerType
	}{
		{"ETB", "Doppelganger", "When ~ enters the battlefield, draw a card", TriggerETB},
		{"Upkeep", "Lord of the Pit", "At the beginning of your upkeep, sacrifice a creature", TriggerUpkeep},
	}

	p := NewParser()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.Parse(tt.cardName, tt.text)
			if len(result.Abilities) == 0 {
				t.Errorf("expected abilities, got none for %q", tt.text)
				return
			}
			ability := result.Abilities[0]
			if ability.Trigger == nil {
				t.Error("expected trigger, got nil")
				return
			}
			if ability.Trigger.Type != tt.triggerType {
				t.Errorf("expected trigger type %v, got %v", tt.triggerType, ability.Trigger.Type)
			}
		})
	}
}

func TestParseMultipleAbilities(t *testing.T) {
	p := NewParser()

	result := p.Parse("Serra Angel", "Flying. Vigilance")
	if len(result.Abilities) != 2 {
		t.Errorf("expected 2 abilities, got %d", len(result.Abilities))
		return
	}
	if !result.HasKeyword[KeywordFlying] {
		t.Error("expected Flying keyword")
	}
	if !result.HasKeyword[KeywordVigilance] {
		t.Error("expected Vigilance keyword")
	}
}

func TestParseShivanDragonFull(t *testing.T) {
	p := NewParser()
	text := "Flying. {R}: Shivan Dragon gets +1/+0 until end of turn"
	result := p.Parse("Shivan Dragon", text)

	if len(result.Abilities) != 2 {
		t.Errorf("expected 2 abilities, got %d", len(result.Abilities))
		return
	}

	if !result.HasKeyword[KeywordFlying] {
		t.Error("expected Flying keyword")
	}

	var foundPump bool
	for _, ability := range result.Abilities {
		if boost, ok := ability.Effect.(*effects.StatBoost); ok {
			foundPump = true
			if boost.PowerBoost != 1 || boost.ToughnessBoost != 0 {
				t.Errorf("expected +1/+0, got +%d/+%d", boost.PowerBoost, boost.ToughnessBoost)
			}
		}
	}
	if !foundPump {
		t.Error("expected pump ability")
	}
}

func TestParseBlackKnight(t *testing.T) {
	p := NewParser()
	text := "First strike. Protection from white"
	result := p.Parse("Black Knight", text)

	if len(result.Abilities) != 2 {
		t.Errorf("expected 2 abilities, got %d", len(result.Abilities))
		return
	}

	if !result.HasKeyword[KeywordFirstStrike] {
		t.Error("expected First Strike keyword")
	}
	if !result.HasKeyword[KeywordProtection] {
		t.Error("expected Protection keyword")
	}
}

func TestParseTargetSpecs(t *testing.T) {
	tests := []struct {
		text       string
		targetType TargetType
		controller Controller
		count      int
	}{
		{"any target", TargetTypeAny, ControllerAny, 1},
		{"target creature", TargetTypeCreature, ControllerAny, 1},
		{"target player", TargetTypePlayer, ControllerAny, 1},
		{"target opponent", TargetTypePlayer, ControllerOpponent, 1},
		{"each creature", TargetTypeCreature, ControllerAny, -1},
		{"all creatures", TargetTypeCreature, ControllerAny, -1},
		{"creatures you control", TargetTypeCreature, ControllerYou, -1},
		{"target land", TargetTypeLand, ControllerAny, 1},
		{"target artifact", TargetTypeArtifact, ControllerAny, 1},
		{"enchanted creature", TargetTypeCreature, ControllerAny, 1},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			spec := ParseTarget(tt.text)
			if spec == nil {
				t.Errorf("expected target spec for %q", tt.text)
				return
			}
			if spec.Type != tt.targetType {
				t.Errorf("expected type %v, got %v", tt.targetType, spec.Type)
			}
			if spec.Controller != tt.controller {
				t.Errorf("expected controller %v, got %v", tt.controller, spec.Controller)
			}
			if spec.Count != tt.count {
				t.Errorf("expected count %d, got %d", tt.count, spec.Count)
			}
		})
	}
}

func TestParseCosts(t *testing.T) {
	tests := []struct {
		input    string
		hasTap   bool
		mana     string
	}{
		{"{T}", true, ""},
		{"{G}", false, "{G}"},
		{"{1}{G}", false, "{1}{G}"},
		{"{T}{G}", true, "{G}"},
		{"{2}{R}{R}", false, "{2}{R}{R}"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cost := ParseCost(tt.input)
			if cost == nil {
				t.Error("expected cost, got nil")
				return
			}
			if cost.Tap != tt.hasTap {
				t.Errorf("expected tap %v, got %v", tt.hasTap, cost.Tap)
			}
			if cost.Mana != tt.mana {
				t.Errorf("expected mana %q, got %q", tt.mana, cost.Mana)
			}
		})
	}
}

func TestPreprocessCardNameReplacement(t *testing.T) {
	p := NewParser()

	result := p.Parse("Prodigal Sorcerer", "{T}: ~ deals 1 damage to any target")
	if len(result.Abilities) == 0 {
		t.Error("expected ability after ~ replacement")
		return
	}

	ability := result.Abilities[0]
	dmg, ok := ability.Effect.(*effects.DirectDamage)
	if !ok {
		t.Errorf("expected DirectDamage, got %T", ability.Effect)
		return
	}
	if dmg.Amount != 1 {
		t.Errorf("expected 1 damage, got %d", dmg.Amount)
	}
}
