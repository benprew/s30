package core_engine

import "github.com/benprew/s30/mtg/core_engine/events"

// - effects - effects are an interface to the different types of actions that can
//   happen in a game. Things like dealing direct damage to a target, casting a
//   creature, drawing a card, using an ability of a card in play. These are generic
//   effects that can encompass many actions that could be taken by different cards.
//
// Examples:
// - dealing damage to a target (bolt, shock, drain life, chain lightning)
//
// A given card may have multiple effects as part of it's resolution (drain life
// deals damage and gains life)
//
// An action can contain multiple effects? Should that be an action or a card? Or
// should "drain life" be a single effect that just calls deal damage and gain
// life?

// When casting a spell, target should be chosen at cast time
// Some spells may require the state of the board at resolution time? Maybe sm
// should return events so game state can resolve them?

// Each object on the battlefield, graveyard, hands, etc must have a unique
// identifier so they can be targeted

type Event interface {
    Name() string
    Resolve()
    Target() events.Targetable
    TargetType() string
}
