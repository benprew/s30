package domain

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/benprew/s30/assets"
)

// QuestRewardTier scales the reward a deck-changing quest pays out.
type QuestRewardTier int

const (
	// TierStandard is the easy passive (e.g. play lands): gold only.
	TierStandard QuestRewardTier = iota
	// TierThemed is a multi-duel action grind (cast N color/type spells): the
	// top payout, since it takes several games to complete.
	TierThemed
	// TierChallenge is a single-duel deck-constraint win: solid gold plus a card,
	// but less than a themed grind.
	TierChallenge
)

type questDefRaw struct {
	ID           string `toml:"id"`
	Type         string `toml:"type"`
	Title        string `toml:"title"`
	Description  string `toml:"description"`
	DeadlineDays int    `toml:"deadline_days"`
	RewardTier   string `toml:"reward_tier"`
	Metric       string `toml:"metric"`
	Color        string `toml:"color"`
	CardType     string `toml:"card_type"`
	Target       int    `toml:"target"`
	Constraint   string `toml:"constraint"`
	N            int    `toml:"n"`
}

type questDefFile struct {
	Quests []questDefRaw `toml:"quest"`
}

// QuestDefs is the loaded pool of deck-changing quest templates, keyed by ID.
var QuestDefs = loadQuestDefs()

// QuestDefList returns the quest templates sorted by ID for stable iteration.
func QuestDefList() []*Quest {
	defs := make([]*Quest, 0, len(QuestDefs))
	for _, d := range QuestDefs {
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs
}

func loadQuestDefs() map[string]*Quest {
	defs := make(map[string]*Quest)
	configDir := "configs/quests"

	files, err := assets.QuestCfgFS.ReadDir(configDir)
	if err != nil {
		panic(fmt.Errorf("error reading quest configs: %w", err))
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".toml") {
			continue
		}
		data, err := assets.QuestCfgFS.ReadFile(path.Join(configDir, f.Name()))
		if err != nil {
			panic(fmt.Errorf("error reading embedded %s: %w", f.Name(), err))
		}
		var file questDefFile
		if _, err := toml.Decode(string(data), &file); err != nil {
			panic(fmt.Errorf("error decoding embedded %s: %w", f.Name(), err))
		}
		for i := range file.Quests {
			def, err := parseQuestDef(&file.Quests[i])
			if err != nil {
				panic(fmt.Errorf("invalid quest in %s: %w", f.Name(), err))
			}
			if _, dup := defs[def.ID]; dup {
				panic(fmt.Errorf("duplicate quest id %q in %s", def.ID, f.Name()))
			}
			defs[def.ID] = def
		}
	}
	return defs
}

func parseQuestDef(raw *questDefRaw) (*Quest, error) {
	if raw.ID == "" {
		return nil, fmt.Errorf("quest is missing id")
	}
	def := &Quest{
		ID:           raw.ID,
		Title:        raw.Title,
		Description:  raw.Description,
		DeadlineDays: raw.DeadlineDays,
		Target:       raw.Target,
		ConstraintN:  raw.N,
		Color:        ParseColorMask(raw.Color),
		CardTypes:    parseCardTypes(raw.CardType),
	}

	tier, err := parseRewardTier(raw.RewardTier)
	if err != nil {
		return nil, err
	}
	def.RewardTier = tier

	switch strings.ToLower(raw.Type) {
	case "action", "action_tracker", "tracker":
		def.Type = QuestTypeActionTracker
		metric, err := parseMetric(raw.Metric)
		if err != nil {
			return nil, err
		}
		def.Metric = metric
		if def.Target <= 0 {
			return nil, fmt.Errorf("action quest %q needs a positive target", raw.ID)
		}
	case "constraint", "deck_constraint", "deck":
		def.Type = QuestTypeDeckConstraint
		constraint, err := parseConstraint(raw.Constraint)
		if err != nil {
			return nil, err
		}
		def.Constraint = constraint
	default:
		return nil, fmt.Errorf("quest %q has unknown type %q", raw.ID, raw.Type)
	}

	if def.DeadlineDays <= 0 {
		def.DeadlineDays = 20
	}
	return def, nil
}

func parseMetric(s string) (QuestMetric, error) {
	switch strings.ToLower(s) {
	case "cast_color":
		return MetricCastColor, nil
	case "play_lands":
		return MetricPlayLands, nil
	case "attack_creatures":
		return MetricAttackCreatures, nil
	case "destroy_enemy_creatures":
		return MetricDestroyEnemyCreatures, nil
	case "cast_type":
		return MetricCastType, nil
	case "direct_damage":
		return MetricDirectDamage, nil
	default:
		return MetricNone, fmt.Errorf("unknown metric %q", s)
	}
}

func parseConstraint(s string) (QuestConstraint, error) {
	switch strings.ToLower(s) {
	case "mono_color":
		return ConstraintMonoColor, nil
	case "fat_deck":
		return ConstraintFatDeck, nil
	case "low_curve":
		return ConstraintLowCurve, nil
	case "color_light":
		return ConstraintColorLight, nil
	case "no_attacking":
		return ConstraintNoAttacking, nil
	default:
		return ConstraintNone, fmt.Errorf("unknown constraint %q", s)
	}
}

func parseRewardTier(s string) (QuestRewardTier, error) {
	switch strings.ToLower(s) {
	case "", "standard":
		return TierStandard, nil
	case "themed":
		return TierThemed, nil
	case "challenge":
		return TierChallenge, nil
	default:
		return TierStandard, fmt.Errorf("unknown reward tier %q", s)
	}
}

// parseCardTypes parses a comma- or space-separated card-type spec
// (e.g. "instant,sorcery") into a slice of CardType, dropping unknowns.
func parseCardTypes(s string) []CardType {
	s = strings.NewReplacer(",", " ", "|", " ").Replace(s)
	var types []CardType
	for field := range strings.FieldsSeq(s) {
		if t := parseCardType(field); t != "" {
			types = append(types, t)
		}
	}
	return types
}

// ParseColorMask parses a color spec into a ColorMask. It accepts concatenated
// single-letter WUBRG codes ("R", "RB", "wubrg") and full color names
// ("red", "black"), case-insensitive. An empty/unknown spec yields ColorColorless.
func ParseColorMask(s string) ColorMask {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ColorColorless
	}
	switch s {
	case "white":
		return ColorWhite
	case "blue":
		return ColorBlue
	case "black":
		return ColorBlack
	case "red":
		return ColorRed
	case "green":
		return ColorGreen
	case "any", "all":
		return ColorAny
	}
	var m ColorMask
	for _, r := range s {
		switch r {
		case 'w':
			m |= ColorWhite
		case 'u':
			m |= ColorBlue
		case 'b':
			m |= ColorBlack
		case 'r':
			m |= ColorRed
		case 'g':
			m |= ColorGreen
		}
	}
	return m
}
