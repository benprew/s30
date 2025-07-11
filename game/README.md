```
# Game

This is all the code for the Shanalar game. This the is UI the player sees and the maps they walk around on. The game engine and cards are separate.

# Package Structure


```
game/
├── entities/
├── minimap/
├── screens/       <-- Might move some screen logic here or refactor
├── sprites/       <-- For sprite sheet loading and raw sprite assets
│   └── loader.go
├── ui/            <-- New directory for UI organization
│   ├── elements/  <-- Reusable UI components (e.g., Button)
│   │   └── button.go
│   ├── fonts/     <-- Font loading and text drawing utilities
│   │   └── text.go    <-- Contains DrawText
│   ├── screens/   <-- Screen management/composition (optional, could stay in game/screens)
│   └── utils.go   <-- General UI utilities
├── world/
├── Game.go
└── ...
```
