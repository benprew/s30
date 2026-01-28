# MTG Rules Engine

This directory contains the Magic: The Gathering rules engine implementing the stack, phases, combat, and mana systems.

## Testing

Run tests from the project root:
```bash
make test
```

## Package Structure

```
mtg/
├── effects/          # Card abilities and effects (Event implementations)
│   ├── interfaces.go # Event + Targetable interfaces
│   ├── direct_damage.go
│   ├── stat_boost.go
│   └── ability.go
└── core/             # Game rules (stack, phases, combat, mana, players)
    ├── game.go
    ├── stack.go
    ├── turn.go
    ├── combat.go
    ├── player.go
    ├── card.go
    ├── mana.go
    ├── actions.go
    └── interaction.go
```

## Architecture

### State Machines

**Turn Phases** (`core/turn.go`):
- Untap → Upkeep → Draw → Main1 → Combat → Main2 → End → Cleanup

**Combat Steps**:
- BeginningOfCombat → DeclareAttackers → DeclareBlockers → CombatDamage → EndOfCombat

**Stack** (`core/stack.go`):
- States: StartStack → WaitPlayer → Resolve → Empty
- Implements two-player priority passing per MTG rule 117.3

### Key Files

| Package | File | Purpose |
|---------|------|---------|
| `core` | `game.go` | Main game loop, turn phases, action processing |
| `core` | `stack.go` | Stack state machine, priority system |
| `core` | `combat.go` | Attack/block declarations, damage resolution |
| `core` | `mana.go` | Mana pool, cost parsing (`{3}{G}{R}`), payment |
| `core` | `card.go` | Card state, power/toughness, zone tracking |
| `core` | `player.go` | Life, zones (hand, library, battlefield, graveyard) |
| `effects` | `interfaces.go` | Event and Targetable interfaces |
| `effects` | `direct_damage.go` | DirectDamage effect implementation |
| `effects` | `stat_boost.go` | StatBoost effect implementation |

### Adding New Card Effects

1. Create a new file in `effects/` implementing the `Event` interface (see `direct_damage.go`, `stat_boost.go`)
2. Use the `Targetable` interface methods (`ReceiveDamage`, `AddPowerBoost`, `AddToughnessBoost`) instead of type assertions
3. Add the action mapping in `core/card.go` `CardActions()` function
4. Add tests in the corresponding `_test.go` file

## Code Style

- Add comments to document why and not what. Limit comments to structs and public functions
- Make code self-explanatory
- Run `golangci-lint run` after changes
- Use async player input via `InputChan` for both human and AI players
