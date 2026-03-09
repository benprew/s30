# Sound System Design Doc

## Overview

Add audio (SFX + music) to Shandalar 30 using Ebiten's built-in `ebiten/v2/audio` package. No new dependencies needed.

## Architecture

### Audio Manager

A single `AudioManager` struct, initialized once in `game.NewGame()` and passed to screens that need it.

```go
// game/audio/manager.go
type AudioManager struct {
    context    *audio.Context
    bgmPlayer *audio.Player   // current background music (looping)
    sfxCache   map[SFX]*audio.Player
    bgmVolume  float64
    sfxVolume  float64
    muted      bool
}
```

Key methods:

- `PlaySFX(sfx SFX)` -- fire-and-forget, plays from cache
- `PlayBGM(bgm BGM)` -- crossfade/switch looping background music
- `StopBGM()` -- fade out and stop
- `SetVolume(bgm, sfx float64)` -- separate volume controls
- `Mute() / Unmute()`

### Asset Embedding

Follow the existing pattern in `assets/embed.go`. Create `assets/audio/sfx/` and `assets/audio/bgm/` directories.

```go
//go:embed audio/sfx/*
var SfxFS embed.FS

//go:embed audio/bgm/*
var BgmFS embed.FS
```

### WAV Decoding

Use Ebiten's `audio/wav` package:

```go
import "github.com/hajimehoshi/ebiten/v2/audio/wav"

stream, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
```

Ebiten's audio context requires a consistent sample rate (typically 44100 or 48000 Hz). All `.wav` files must use the same sample rate.

## Sound Events

### Priority 1 -- High Impact

| Event | Sound | Type | Trigger Location |
|---|---|---|---|
| Button click | UI click | SFX | `elements/button.go` -- `IsClicked()` |
| Duel start | Battle horn | SFX | `duel_ante.go` -- start duel button |
| Duel won | Victory fanfare | SFX | `duel_win.go` -- on enter |
| Duel lost | Defeat sting | SFX | `duel_lose.go` -- on enter |
| Spell cast | Cast sound | SFX | `duel.go` -- when spell goes on stack |
| Combat damage | Hit sound | SFX | `duel.go` -- damage resolution |
| Creature death | Death sound | SFX | `mtg/core` -- creature dies |
| World BGM | Exploration music | BGM | `level.go` -- on enter |
| Duel BGM | Battle music | BGM | `duel.go` -- on enter |

### Priority 2 -- Polish

| Event | Sound | Type |
|---|---|---|
| Enemy encounter (world) | Alert sting | SFX |
| Enter/exit city | Door/gate sound | SFX |
| Play a land | Land drop thud | SFX |
| Tap mana | Mana chime | SFX |
| Card draw | Card draw swoosh | SFX |
| Phase change | Subtle tick | SFX |
| City BGM | Town music | BGM |
| Menu screen BGM | Title music | BGM |

### Priority 3 -- Flavor

| Event | Sound | Type |
|---|---|---|
| Color-specific cast sounds | Per mana color | SFX |
| Life gain / life loss | Distinct sounds | SFX |
| Creature attack declaration | War cry | SFX |
| Counter spell | Fizzle | SFX |
| Ambient world sounds | Birds, wind | BGM layer |

## File Structure

```
assets/
  audio/
    sfx/
      click.wav
      cast.wav
      damage.wav
      death.wav
      victory.wav
      defeat.wav
      encounter.wav
      card_draw.wav
      land_play.wav
    bgm/
      world.wav
      battle.wav
      city.wav
      title.wav
```

## Integration Points

### 1. Pass AudioManager through screens

The `Game` struct already holds all screens. Add `AudioManager` as a field and pass it during screen construction, same pattern as existing shared state.

### 2. SFX triggers in screens

Screens call `audioManager.PlaySFX(audio.SFXClick)` at event points. Minimal code change -- one line per event.

### 3. BGM triggers on screen transitions

In `game.go`'s `Update()`, when the active screen changes, call the appropriate `PlayBGM()` for the new screen.

### 4. MTG engine events

The MTG engine (`mtg/core`) is decoupled from the UI. Two options:

- **Option A (simple):** The `DuelScreen` checks game state changes each `Update()` and triggers sounds. Keeps audio out of the engine.
- **Option B (event system):** Add a callback/channel to the engine that emits events. More flexible but more invasive.

**Recommendation: Option A.** The duel screen already reads game state every frame. Add comparisons (e.g., "did a creature die since last frame?") to trigger sounds.

## Practical Notes

### WAV files

- All files must use the **same sample rate** (44100 Hz recommended)
- Keep SFX short (< 2 seconds) for responsiveness
- For BGM, `.wav` works but is large. Consider converting to **Ogg Vorbis** (`.ogg`) for smaller binaries -- Ebiten supports it via `audio/vorbis`. This matters since assets are embedded in the binary.

### Verifying and converting sample rates

```bash
# Check sample rate
ffprobe yourfile.wav

# Convert to 44100 Hz
ffmpeg -i input.wav -ar 44100 output.wav

# Convert to ogg for BGM (smaller file size)
ffmpeg -i input.wav -c:a libvorbis -q:a 4 output.ogg
```

### Performance

- Game runs at TPS=10, so audio is not performance-sensitive
- Pre-decode and cache all SFX at startup
- BGM can stream from embedded FS

### Volume defaults

- SFX: 70%, BGM: 40% -- music shouldn't overpower effects
- Add a settings screen later, or keyboard shortcuts (M to mute) as a quick win

## Implementation Order

1. Create `game/audio/` package with `AudioManager`
2. Add embed entries in `assets/embed.go`
3. Wire `AudioManager` into `Game` struct
4. Add button click SFX (proves the pipeline works end-to-end)
5. Add BGM for world and duel screens
6. Add remaining Priority 1 SFX
7. Polish with Priority 2/3 as desired
