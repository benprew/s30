package parser

import (
	"regexp"
	"strings"
)

type TargetType string

const (
	TargetTypeCreature  TargetType = "creature"
	TargetTypePlayer    TargetType = "player"
	TargetTypeAny       TargetType = "any"
	TargetTypePermanent TargetType = "permanent"
	TargetTypeLand      TargetType = "land"
	TargetTypeArtifact  TargetType = "artifact"
	TargetTypeEnchant   TargetType = "enchantment"
	TargetTypeSpell     TargetType = "spell"
)

type Controller string

const (
	ControllerYou      Controller = "you"
	ControllerOpponent Controller = "opponent"
	ControllerAny      Controller = "any"
)

type TargetSpec struct {
	Type       TargetType
	Controller Controller
	Subtype    string
	Count      int
	Condition  string
}

var targetPatterns = []struct {
	regex *regexp.Regexp
	parse func([]string) *TargetSpec
}{
	{
		regexp.MustCompile(`(?i)any target`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeAny, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target (attacking|blocking|tapped|untapped)?\s*creature`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerAny, Count: 1, Condition: m[1]}
		},
	},
	{
		regexp.MustCompile(`(?i)target player`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypePlayer, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target opponent`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypePlayer, Controller: ControllerOpponent, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)each creature`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerAny, Count: -1}
		},
	},
	{
		regexp.MustCompile(`(?i)all creatures`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerAny, Count: -1}
		},
	},
	{
		regexp.MustCompile(`(?i)creatures you control`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerYou, Count: -1}
		},
	},
	{
		regexp.MustCompile(`(?i)target (\w+) creature`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerAny, Count: 1, Subtype: m[1]}
		},
	},
	{
		regexp.MustCompile(`(?i)target creature or player`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeAny, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target land`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeLand, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target permanent`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypePermanent, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target artifact`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeArtifact, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)target enchantment`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeEnchant, Controller: ControllerAny, Count: 1}
		},
	},
	{
		regexp.MustCompile(`(?i)enchanted creature`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeCreature, Controller: ControllerAny, Count: 1, Condition: "enchanted"}
		},
	},
	{
		regexp.MustCompile(`(?i)enchanted land`),
		func(m []string) *TargetSpec {
			return &TargetSpec{Type: TargetTypeLand, Controller: ControllerAny, Count: 1, Condition: "enchanted"}
		},
	},
}

func ParseTarget(text string) *TargetSpec {
	text = strings.ToLower(text)
	for _, p := range targetPatterns {
		matches := p.regex.FindStringSubmatch(text)
		if matches != nil {
			return p.parse(matches)
		}
	}
	return nil
}
