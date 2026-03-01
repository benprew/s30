# MTG Rules Engine

This directory contains the Magic: The Gathering rules engine implementing the stack, phases, combat, mana, card text parsing, and AI systems.

## Testing

Run tests from the project root:
```bash
make test
```

## Package Structure

```
mtg/
├── core/             # Game rules (stack, phases, combat, mana, players)
│   ├── game.go       # Main game loop, turn phases, action processing
│   ├── stack.go      # Stack state machine, priority system
│   ├── turn.go       # Phase/combat step enums
│   ├── combat.go     # Attack/block, damage (deathtouch, trample, lifelink, first strike)
│   ├── player.go     # Life, zones (hand, library, battlefield, graveyard, exile)
│   ├── card.go       # Card state, P/T with boosts, aura attachments, keywords
│   ├── mana.go       # Mana pool (supports dual lands), cost parsing, auto-tapping
│   ├── actions.go    # PlayLand, CastSpell, ActivateManaAbility
│   └── interaction.go
├── effects/          # Card abilities and effects (Event implementations)
│   ├── interfaces.go # Event + Targetable interfaces
│   ├── keywords.go   # 29 keyword abilities (Flying, Trample, Haste, etc.)
│   ├── direct_damage.go
│   ├── stat_boost.go
│   ├── lord_effect.go # Static creature type bonuses (+X/+Y to Goblins, etc.)
│   ├── mana_ability.go
│   └── ability.go
├── parser/           # Card ability text parsing
│   ├── parser.go     # Regex-based pattern matching engine
│   ├── patterns.go   # Pattern registrations (keywords, damage, auras, lords, etc.)
│   ├── costs.go      # Mana/tap/sacrifice cost parsing
│   └── targets.go    # Target specification parsing (creature, player, controller)
└── ai/               # AI decision making
    └── ai.go         # Action selection + goroutine-based AI runner
```

## Architecture

### Key Interfaces

**Targetable** (`effects/interfaces.go`) — implemented by Card and Player:
- `ReceiveDamage(int)`, `IsDead() bool`, `AddPowerBoost(int)`, `AddToughnessBoost(int)`

**Event** (`effects/interfaces.go`) — implemented by all effects:
- `Resolve()`, `Target() Targetable`, `AddTarget(Targetable)`

### State Machines

**Turn Phases** (`core/turn.go`):
- Untap → Upkeep → Draw → Main1 → Combat → Main2 → End → Cleanup

**Combat Steps** (`core/combat.go`):
- BeginningOfCombat → DeclareAttackers → DeclareBlockers → FirstStrikeDamage → CombatDamage → EndOfCombat

**Stack** (`core/stack.go`):
- States: StartStack → WaitPlayer → Resolve → Empty
- Implements two-player priority passing per MTG rule 117.3

### Card Text Parser

The parser (`parser/`) converts card text into structured `ParsedAbility` objects:
1. Text is preprocessed (~ replaced with card name, reminder text stripped)
2. Split into sentences and matched against registered regex patterns
3. Each pattern handler creates the appropriate effect (DirectDamage, StatBoost, etc.)
4. Unparsed sentences are captured for debugging (`cmd/parse_cards --unparsed`)

### AI System

The AI (`ai/`) runs as a goroutine per AI player:
- Waits on `player.WaitingChan` for priority
- Selects actions via priority: Discard > Cast > Play Land > Attack > Block > Pass
- Sends chosen action to `player.InputChan`

### Adding New Card Effects

1. Create a struct in `effects/` implementing the `Event` interface
2. Register a pattern in `parser/patterns.go` via `RegisterPattern()`
3. The pattern handler creates your effect from regex matches
4. Add special cases in `core/card.go` `CardActions()` if needed
5. Add tests in the corresponding `_test.go` file

### Adding New Keywords

1. Add `KeywordXxx Keyword = "xxx"` to `effects/keywords.go`
2. Add to `KeywordMap` in the same file
3. Register a pattern in `parser/patterns.go` `registerKeywordPatterns()`

## Code Style

- Comments explain why, not what. Limit to structs and public functions
- Make code self-explanatory
- Run `golangci-lint run` after changes
- Use async player input via `InputChan` for both human and AI players
