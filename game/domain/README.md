# Domain Objects

Contains all the info on objecs in the game.

This is where we store the representations of all the objects in the game, cities, players, rogues, cards, etc.

These objects should never refer to display parts of the game (ie screens or level) to prevent import loops.

## Characters

Structs and locations
- character.go:Character - common character traits between players and enemies
- character.go:CharacterInstance - character location/speed/etc
- rogue.go - loads a map of Name -> Character struct from .toml files. Rogues are R/O
- enemy.go:Enemy - instance of a Rogue, union of Character and CharacterInstance structs
- player_character.go:Player - your character, has player-specific fields like gold, food, cards, etc

There are two types of characters, derived from Character:
- Enemies - an instance of a rogue. Rogues are loaded from the .toml files in assets/config/rouges/
- Player - adds additional data such as food, gold, their card collection, location, world magics, etc

## Appendix

In programming, domain objects are classes, structs, or data types that represent concepts from the "problem domain" (the real-world area your program is modeling).

They encapsulate data and behavior directly related to those concepts.
