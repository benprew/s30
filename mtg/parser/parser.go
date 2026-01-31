package parser

import (
	"regexp"
	"strings"

	"github.com/benprew/s30/mtg/effects"
)

type AbilityType string

const (
	AbilityTypeActivated AbilityType = "Activated"
	AbilityTypeTriggered AbilityType = "Triggered"
	AbilityTypeStatic    AbilityType = "Static"
	AbilityTypeKeyword   AbilityType = "Keyword"
	AbilityTypeSpell     AbilityType = "Spell"
)

type ParsedAbility struct {
	Type       AbilityType
	Cost       *Cost
	Effect     effects.Event
	Keywords   []Keyword
	Trigger    *Trigger
	TargetSpec *TargetSpec
	RawText    string
}

type Trigger struct {
	Type      TriggerType
	Condition string
}

type TriggerType string

const (
	TriggerETB         TriggerType = "EntersTheBattlefield"
	TriggerDeath       TriggerType = "Dies"
	TriggerUpkeep      TriggerType = "Upkeep"
	TriggerCombatDmg   TriggerType = "CombatDamage"
	TriggerCreatureDmg TriggerType = "CreatureDamage"
)

type ParseResult struct {
	Abilities  []*ParsedAbility
	Unparsed   []string
	HasKeyword map[Keyword]bool
}

type Pattern struct {
	Name    string
	Regex   *regexp.Regexp
	Handler func(matches []string, cardName string) (*ParsedAbility, error)
}

type Parser struct {
	patterns []Pattern
}

var DefaultParser = NewParser()

func NewParser() *Parser {
	p := &Parser{
		patterns: make([]Pattern, 0),
	}
	p.registerPatterns()
	return p
}

func (p *Parser) RegisterPattern(name string, regex *regexp.Regexp, handler func([]string, string) (*ParsedAbility, error)) {
	p.patterns = append(p.patterns, Pattern{
		Name:    name,
		Regex:   regex,
		Handler: handler,
	})
}

func (p *Parser) Parse(cardName, text string) ParseResult {
	result := ParseResult{
		Abilities:  make([]*ParsedAbility, 0),
		Unparsed:   make([]string, 0),
		HasKeyword: make(map[Keyword]bool),
	}

	if text == "" {
		return result
	}

	text = p.preprocess(text, cardName)
	sentences := p.splitSentences(text)

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		parsed := false
		for _, pattern := range p.patterns {
			matches := pattern.Regex.FindStringSubmatch(sentence)
			if matches != nil {
				ability, err := pattern.Handler(matches, cardName)
				if err == nil && ability != nil {
					ability.RawText = sentence
					result.Abilities = append(result.Abilities, ability)
					for _, kw := range ability.Keywords {
						result.HasKeyword[kw] = true
					}
					parsed = true
					break
				}
			}
		}

		if !parsed {
			result.Unparsed = append(result.Unparsed, sentence)
		}
	}

	return result
}

func (p *Parser) preprocess(text, cardName string) string {
	text = strings.ReplaceAll(text, "~", cardName)
	text = strings.ReplaceAll(text, "  ", " ")
	return strings.TrimSpace(text)
}

func (p *Parser) splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	inBraces := 0

	for i, r := range text {
		current.WriteRune(r)

		switch r {
		case '{':
			inBraces++
		case '}':
			inBraces--
		}

		if inBraces == 0 {
			if r == '.' || r == '\n' {
				s := strings.TrimSpace(current.String())
				if s != "" && s != "." {
					sentences = append(sentences, strings.TrimSuffix(s, "."))
				}
				current.Reset()
			} else if i < len(text)-1 {
				next := text[i+1]
				if r == ':' && next != ' ' {
					continue
				}
			}
		}
	}

	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, strings.TrimSuffix(remaining, "."))
	}

	return sentences
}
