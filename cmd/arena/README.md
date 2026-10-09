# Arena launcher

Run a standalone arena from its entry screen:

```sh
go run ./cmd/arena
```

The launcher follows the deck editor, opponent screen, duels, rewards, and
champion screen. It exits when you leave the arena or dismiss the loss or
champion screen. It uses a temporary player and does not write a campaign save.

The player starts with 1,000 gold and 10 life. Arena entry costs 150 gold.
You can change starting resources and enable duel debug options:

```sh
go run ./cmd/arena -life 20 -gold 1000 -show-opponent-hand -duel-log
go run ./cmd/arena -champion
```


