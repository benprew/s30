# Shanalar Rules
- **turn state** - state machine that tracks game state. Phase, active player, current players turn, state effects, etc
- **stack state** - state machine that handles the stack.
- **board state** - manages cards moving zones: library, hand, battlefield , gaveyard, exile. Ex. playing lands and creatures. Spells going to graveyard, creatures dying, etc.
- **player state** - tracks player attributes: life, poison counters, whether they've lost the game. (Winning the game is defined as the other player losing. If both players lose at the same time, it's a draw)
- **action** - actions that can occur in the game (cast spell, use ability, etc) and handles their resolution

Every phase generate a list of cards with actions for the player (other player needs to have a list of cards available too)

Maybe this should be a list of actions the AI could iterate through those actions?

What is the equivalent to a ply in mtg? One action sequence? A full half turn? In chess a ply is a half turn, which is only a single action. A single action sequence seems about right (ie cast a spell, respond with instant, respond again, etc)

## Core engine
How to handle input from players? Wait for "continue". Both players need to acknowledge. Playgame should be an input loop, with a channel for each player?

No need to notify player, player can just send events into channel and when core rules needs a response from a player it just waits.

AI should have timeout so it doesn't wait forever. AI should run in a separate loop.

"Brawl" UI waits for input from player, continuously getting available actions from mtg game and displaying them in the UI.

## Plan:
- in UI Update function, UI ask engine for available actions in current game state and presents them to the user.
- User inputs which action they want to perform.
- Update captures that input and passes it to the engine.
- Engine runs and updates game state, which the UI renders (via Render).
- AI opponent is similar to a regular user, but uses synthetic input

## Open Questions
How do I represent waiting for a user? I was thinking of using input/output channels for each player. The game engine would send input to the player as a list of available actions, then the player would respond with one of those actions.

## AI
Identify spells that are idempotent or don't change game state depending on when they're cast. I.e. re-pumping mishras factory, gaining life, etc.

prefer playing spells after combat.

AI would do the same, but would also need to be able to create a copy of the game state to see the outcome of specific actions then min/max to choose the best one.

Drafting strategies:
- https://magic.wizards.com/en/news/feature/drafting-101-understanding-signals-2016-04-12

AI
- https://gamedev.stackexchange.com/questions/33972/how-would-one-approach-developing-an-ai-for-a-trading-card-game/82557#82557


## Phase Questions

- Do all phase triggers go on the stack at once? Or can a player respond to in-between triggers going onto the stack?
- Player chooses order of triggers
- Is Moat an Enchant World? What are all the Enchant World cards? - No, but Enchant World is a thing
- Can stacks create sub-stacks or cause new stacks to be created? ex. Ankh of Mishra after a land is destroyed? Does that damage start a new stack? - no sub-stacks, it's always 1 stack and things go onto it or not, it's not "different stacks each time"
- In Main Phase, if P1 passes priority, can P2 respond?

How to represent triggers that happen after a spell resolves?
- when stack resolves, state goes back to start, but then a trigger can start a new stack?
- Or, when a stack resolves, trigger can immediately start a new stack
- Cleanup shouldn't restart stack, it should create a new one

## Rules

📘 Rule 117.3:

When a player has priority and casts a spell or activates an ability, that player retains priority. That player may cast another spell or activate another ability. Eventually, they will pass priority, giving other players a chance to respond.

603.3b If multiple abilities have triggered since the last time a player received priority, each player, in APNAP order, puts triggered abilities he or she controls on the stack in any order he or she chooses. (See rule 101.4.) Then the game once again checks for and resolves state-based actions until none are performed, then abilities that triggered during this process go on the stack. This process repeats until no new state-based actions are performed and no abilities trigger. Then the appropriate player gets priority.

101.4. If multiple players would make choices and/or take actions at the same time, the active player (the player whose turn it is) makes any choices required, then the next player in turn order (usually the player seated to the active player’s left) makes any choices required, followed by the remaining nonactive players in turn order. Then the actions happen simultaneously. This rule is often referred to as the “Active Player, Nonactive Player (APNAP) order” rule.
