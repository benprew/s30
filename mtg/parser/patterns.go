package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/benprew/s30/mtg/effects"
)

type Keyword = effects.Keyword

const (
	KeywordFlying       = effects.KeywordFlying
	KeywordFirstStrike  = effects.KeywordFirstStrike
	KeywordDoubleStrike = effects.KeywordDoubleStrike
	KeywordTrample      = effects.KeywordTrample
	KeywordHaste        = effects.KeywordHaste
	KeywordVigilance    = effects.KeywordVigilance
	KeywordDeathtouch   = effects.KeywordDeathtouch
	KeywordLifelink     = effects.KeywordLifelink
	KeywordReach        = effects.KeywordReach
	KeywordDefender     = effects.KeywordDefender
	KeywordBanding      = effects.KeywordBanding
	KeywordRampage      = effects.KeywordRampage
	KeywordFear         = effects.KeywordFear
	KeywordShadow       = effects.KeywordShadow
	KeywordHorsemanship = effects.KeywordHorsemanship
	KeywordProtection   = effects.KeywordProtection
	KeywordLandwalk     = effects.KeywordLandwalk
	KeywordShroud       = effects.KeywordShroud
	KeywordFlash        = effects.KeywordFlash
	KeywordRegeneration = effects.KeywordRegeneration
)

func (p *Parser) registerPatterns() {
	p.registerKeywordPatterns()
	p.registerDamagePatterns()
	p.registerStatBoostPatterns()
	p.registerManaAbilityPatterns()
	p.registerActivatedAbilityPatterns()
	p.registerLordPatterns()
	p.registerTriggeredPatterns()
}

func (p *Parser) registerKeywordPatterns() {
	simpleKeywords := []struct {
		name    string
		keyword Keyword
	}{
		{"flying", KeywordFlying},
		{"first strike", KeywordFirstStrike},
		{"double strike", KeywordDoubleStrike},
		{"trample", KeywordTrample},
		{"haste", KeywordHaste},
		{"vigilance", KeywordVigilance},
		{"deathtouch", KeywordDeathtouch},
		{"lifelink", KeywordLifelink},
		{"reach", KeywordReach},
		{"defender", KeywordDefender},
		{"banding", KeywordBanding},
		{"rampage", KeywordRampage},
		{"fear", KeywordFear},
		{"shadow", KeywordShadow},
		{"horsemanship", KeywordHorsemanship},
		{"shroud", KeywordShroud},
		{"flash", KeywordFlash},
	}

	for _, kw := range simpleKeywords {
		keyword := kw.keyword
		p.RegisterPattern(
			kw.name,
			regexp.MustCompile(`(?i)^`+kw.name+`$`),
			func(matches []string, cardName string) (*ParsedAbility, error) {
				return &ParsedAbility{
					Type:     AbilityTypeKeyword,
					Keywords: []Keyword{keyword},
					Effect:   &effects.KeywordAbility{Keywords: []effects.Keyword{keyword}},
				}, nil
			},
		)
	}

	p.RegisterPattern(
		"keyword-list",
		regexp.MustCompile(`(?i)^(flying|first strike|trample|haste|vigilance|banding|fear|reach|shadow|defender),\s*(flying|first strike|trample|haste|vigilance|banding|fear|reach|shadow|defender)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			var keywords []Keyword
			for i := 1; i < len(matches); i++ {
				if kw, ok := effects.KeywordMap[strings.ToLower(matches[i])]; ok {
					keywords = append(keywords, kw)
				}
			}
			return &ParsedAbility{
				Type:     AbilityTypeKeyword,
				Keywords: keywords,
				Effect:   &effects.KeywordAbility{Keywords: keywords},
			}, nil
		},
	)

	p.RegisterPattern(
		"protection",
		regexp.MustCompile(`(?i)^protection from (\w+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			return &ParsedAbility{
				Type:     AbilityTypeKeyword,
				Keywords: []Keyword{KeywordProtection},
				Effect:   &effects.KeywordAbility{Keywords: []effects.Keyword{KeywordProtection}, Modifier: matches[1]},
			}, nil
		},
	)

	landwalkTypes := []string{"swampwalk", "forestwalk", "mountainwalk", "islandwalk", "plainswalk"}
	for _, lwType := range landwalkTypes {
		lwTypeCopy := lwType
		p.RegisterPattern(
			lwType,
			regexp.MustCompile(`(?i)^`+lwType+`$`),
			func(matches []string, cardName string) (*ParsedAbility, error) {
				return &ParsedAbility{
					Type:     AbilityTypeKeyword,
					Keywords: []Keyword{KeywordLandwalk},
					Effect:   &effects.KeywordAbility{Keywords: []effects.Keyword{KeywordLandwalk}, Modifier: lwTypeCopy},
				}, nil
			},
		)
	}

	p.RegisterPattern(
		"regeneration-cost",
		regexp.MustCompile(`(?i)^(\{[^}]+\}(?:\{[^}]+\})*)\s*:\s*regenerate\s+[\w\s]+$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			cost := ParseCost(matches[1])
			return &ParsedAbility{
				Type:     AbilityTypeActivated,
				Cost:     cost,
				Keywords: []Keyword{KeywordRegeneration},
				Effect:   &effects.KeywordAbility{Keywords: []effects.Keyword{KeywordRegeneration}},
			}, nil
		},
	)
}

func (p *Parser) registerDamagePatterns() {
	p.RegisterPattern(
		"damage-any-target",
		regexp.MustCompile(`(?i)^(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+any\s+target$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeSpell,
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypeAny, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"damage-target-creature-or-player",
		regexp.MustCompile(`(?i)^(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+target\s+creature\s+or\s+player$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeSpell,
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypeAny, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"damage-target-creature",
		regexp.MustCompile(`(?i)^(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+target\s+creature$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeSpell,
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypeCreature, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"damage-target-player",
		regexp.MustCompile(`(?i)^(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+target\s+player$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeSpell,
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypePlayer, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"tap-damage-any",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+any\s+target$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeActivated,
				Cost:       &Cost{Tap: true},
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypeAny, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"tap-damage-creature-or-player",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*(?:[\w\s]+\s+)?deals?\s+(\d+)\s+damage\s+to\s+target\s+creature\s+or\s+player$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			amount, _ := strconv.Atoi(matches[1])
			return &ParsedAbility{
				Type:       AbilityTypeActivated,
				Cost:       &Cost{Tap: true},
				Effect:     &effects.DirectDamage{Amount: amount},
				TargetSpec: &TargetSpec{Type: TargetTypeAny, Count: 1},
			}, nil
		},
	)
}

func (p *Parser) registerStatBoostPatterns() {
	p.RegisterPattern(
		"target-creature-gets",
		regexp.MustCompile(`(?i)^target\s+creature\s+gets?\s+([+-]?\d+)/([+-]?\d+)(?:\s+until\s+end\s+of\s+turn)?$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			power, _ := strconv.Atoi(matches[1])
			toughness, _ := strconv.Atoi(matches[2])
			return &ParsedAbility{
				Type:       AbilityTypeSpell,
				Effect:     &effects.StatBoost{PowerBoost: power, ToughnessBoost: toughness},
				TargetSpec: &TargetSpec{Type: TargetTypeCreature, Count: 1},
			}, nil
		},
	)

	p.RegisterPattern(
		"enchanted-creature-gets",
		regexp.MustCompile(`(?i)^enchanted\s+creature\s+gets?\s+([+-]?\d+)/([+-]?\d+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			power, _ := strconv.Atoi(matches[1])
			toughness, _ := strconv.Atoi(matches[2])
			return &ParsedAbility{
				Type:       AbilityTypeStatic,
				Effect:     &effects.StatBoost{PowerBoost: power, ToughnessBoost: toughness},
				TargetSpec: &TargetSpec{Type: TargetTypeCreature, Count: 1, Condition: "enchanted"},
			}, nil
		},
	)

	p.RegisterPattern(
		"activated-pump",
		regexp.MustCompile(`(?i)^(\{[^}]+\}(?:\{[^}]+\})*)\s*:\s*(?:[\w\s]+\s+)?gets?\s+([+-]?\d+)/([+-]?\d+)(?:\s+until\s+end\s+of\s+turn)?$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			cost := ParseCost(matches[1])
			power, _ := strconv.Atoi(matches[2])
			toughness, _ := strconv.Atoi(matches[3])
			return &ParsedAbility{
				Type:   AbilityTypeActivated,
				Cost:   cost,
				Effect: &effects.StatBoost{PowerBoost: power, ToughnessBoost: toughness},
			}, nil
		},
	)
}

func (p *Parser) registerManaAbilityPatterns() {
	p.RegisterPattern(
		"tap-add-mana",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*add\s+(\{[WUBRGC]\})$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			symbols := ParseManaSymbols(matches[1])
			return &ParsedAbility{
				Type:   AbilityTypeMana,
				Cost:   &Cost{Tap: true},
				Effect: &effects.ManaAbility{ManaTypes: symbols},
			}, nil
		},
	)

	p.RegisterPattern(
		"tap-add-any-color",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*add\s+one\s+mana\s+of\s+any\s+color$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			return &ParsedAbility{
				Type:   AbilityTypeMana,
				Cost:   &Cost{Tap: true},
				Effect: &effects.ManaAbility{ManaTypes: []string{"W", "U", "B", "R", "G"}, AnyColor: true},
			}, nil
		},
	)

	p.RegisterPattern(
		"tap-add-mana-or",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*add\s+(\{[WUBRGC]\})\s+or\s+(\{[WUBRGC]\})$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			var symbols []string
			for i := 1; i < len(matches); i++ {
				symbols = append(symbols, ParseManaSymbols(matches[i])...)
			}
			return &ParsedAbility{
				Type:   AbilityTypeMana,
				Cost:   &Cost{Tap: true},
				Effect: &effects.ManaAbility{ManaTypes: symbols},
			}, nil
		},
	)

	p.RegisterPattern(
		"tap-add-multiple-mana",
		regexp.MustCompile(`(?i)^\{T\}\s*:\s*add\s+(\{[WUBRGC]\})(\{[WUBRGC]\})$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			var symbols []string
			for i := 1; i < len(matches); i++ {
				symbols = append(symbols, ParseManaSymbols(matches[i])...)
			}
			return &ParsedAbility{
				Type:   AbilityTypeMana,
				Cost:   &Cost{Tap: true},
				Effect: &effects.ManaAbility{ManaTypes: symbols},
			}, nil
		},
	)
}

func (p *Parser) registerActivatedAbilityPatterns() {
	p.RegisterPattern(
		"generic-activated",
		regexp.MustCompile(`(?i)^(\{[^}]+\}(?:\{[^}]+\})*)\s*:\s*(.+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			cost := ParseCost(matches[1])
			effectText := matches[2]

			targetSpec := ParseTarget(effectText)

			return &ParsedAbility{
				Type:       AbilityTypeActivated,
				Cost:       cost,
				TargetSpec: targetSpec,
				RawText:    effectText,
			}, nil
		},
	)
}

func (p *Parser) registerLordPatterns() {
	p.RegisterPattern(
		"other-type-gets-and-has",
		regexp.MustCompile(`(?i)^other\s+(\w+)(?:\s+creatures?)?\s+(?:you\s+control\s+)?get\s+([+-]?\d+)/([+-]?\d+)\s+and\s+have\s+(\w+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			power, _ := strconv.Atoi(matches[2])
			toughness, _ := strconv.Atoi(matches[3])
			keyword := matches[4]
			var kw effects.Keyword
			if k, ok := effects.KeywordMap[strings.ToLower(keyword)]; ok {
				kw = k
			}
			return &ParsedAbility{
				Type: AbilityTypeStatic,
				Effect: &effects.LordEffect{
					Subtype:         matches[1],
					PowerBoost:      power,
					ToughnessBoost:  toughness,
					GrantedKeyword:  &kw,
					GrantedModifier: strings.ToLower(keyword),
					ExcludeSelf:     true,
				},
			}, nil
		},
	)

	p.RegisterPattern(
		"other-type-gets",
		regexp.MustCompile(`(?i)^other\s+(\w+)(?:\s+creatures?)?\s+(?:you\s+control\s+)?get\s+([+-]?\d+)/([+-]?\d+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			power, _ := strconv.Atoi(matches[2])
			toughness, _ := strconv.Atoi(matches[3])
			return &ParsedAbility{
				Type: AbilityTypeStatic,
				Effect: &effects.LordEffect{
					Subtype:        matches[1],
					PowerBoost:     power,
					ToughnessBoost: toughness,
					ExcludeSelf:    true,
				},
			}, nil
		},
	)

	p.RegisterPattern(
		"creatures-you-control-get",
		regexp.MustCompile(`(?i)^creatures?\s+you\s+control\s+get\s+([+-]?\d+)/([+-]?\d+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			power, _ := strconv.Atoi(matches[1])
			toughness, _ := strconv.Atoi(matches[2])
			return &ParsedAbility{
				Type: AbilityTypeStatic,
				Effect: &effects.LordEffect{
					PowerBoost:     power,
					ToughnessBoost: toughness,
					ExcludeSelf:    false,
				},
			}, nil
		},
	)
}

func (p *Parser) registerTriggeredPatterns() {
	p.RegisterPattern(
		"etb-trigger",
		regexp.MustCompile(`(?i)^when\s+[\w\s]+\s+enters\s+the\s+battlefield,?\s+(.+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			effectText := matches[1]
			targetSpec := ParseTarget(effectText)
			return &ParsedAbility{
				Type:       AbilityTypeTriggered,
				Trigger:    &Trigger{Type: TriggerETB},
				TargetSpec: targetSpec,
				RawText:    effectText,
			}, nil
		},
	)

	p.RegisterPattern(
		"whenever-deals-damage",
		regexp.MustCompile(`(?i)^whenever\s+(?:a\s+)?creature\s+dealt\s+damage\s+by\s+[\w\s]+\s+(?:this\s+turn\s+)?dies,?\s+(.+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			effectText := matches[1]
			return &ParsedAbility{
				Type:    AbilityTypeTriggered,
				Trigger: &Trigger{Type: TriggerCreatureDmg, Condition: "dies"},
				RawText: effectText,
			}, nil
		},
	)

	p.RegisterPattern(
		"whenever-attacks",
		regexp.MustCompile(`(?i)^whenever\s+[\w\s]+\s+attacks,?\s+(.+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			effectText := matches[1]
			targetSpec := ParseTarget(effectText)
			return &ParsedAbility{
				Type:       AbilityTypeTriggered,
				Trigger:    &Trigger{Type: TriggerCombatDmg, Condition: "attacks"},
				TargetSpec: targetSpec,
				RawText:    effectText,
			}, nil
		},
	)

	p.RegisterPattern(
		"upkeep-trigger",
		regexp.MustCompile(`(?i)^at\s+the\s+beginning\s+of\s+(your|each)\s+upkeep,?\s+(.+)$`),
		func(matches []string, cardName string) (*ParsedAbility, error) {
			effectText := matches[2]
			targetSpec := ParseTarget(effectText)
			return &ParsedAbility{
				Type:       AbilityTypeTriggered,
				Trigger:    &Trigger{Type: TriggerUpkeep, Condition: matches[1]},
				TargetSpec: targetSpec,
				RawText:    effectText,
			}, nil
		},
	)
}
