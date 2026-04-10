package audio

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"strings"

	"github.com/benprew/s30/assets"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
)

const sampleRate = 22050

// SFX identifies a sound effect.
type SFX int

const (
	SFXClick SFX = iota
	SFXCast
	SFXDamage
	SFXDeath
	SFXVictory
	SFXDefeat
	SFXEncounter
	SFXCardDraw
	SFXLandPlay
	SFXCounter
	SFXCreatureDeath

	// Enemy encounter sounds
	SFXEnemyDragon
	SFXEnemyKnight
	SFXEnemyWolf
	SFXEnemyTroll
	SFXEnemyDjinn
	SFXEnemyWyrm
	SFXEnemyArchmage
	SFXEnemyLord
	SFXEnemyHorse
	SFXEnemyFlying

	// Wizard dialog
	SFXWizardFemale
	SFXWizardMale

	// Rewards and discoveries
	SFXFindCard
	SFXTreasure
	SFXReward
	SFXManaball
	SFXScroll
	SFXNewsflash
	SFXManalink
	SFXDice
	SFXSummon

	// Game state
	SFXWinGame
	SFXStatsScreen

	// World magic (color-specific)
	SFXWorldMagicWhite
	SFXWorldMagicBlue
	SFXWorldMagicBlack
	SFXWorldMagicRed
	SFXWorldMagicGreen

	// Castle entry (color-specific)
	SFXCastleDefault
	SFXCastleBlue
	SFXCastleBlack
	SFXCastleRed
	SFXCastleGreen
)

var sfxFiles = map[SFX]string{
	SFXClick:         "audio/sfx/click.ogg",
	SFXCast:          "audio/sfx/cast.ogg",
	SFXDamage:        "audio/sfx/damage.ogg",
	SFXDeath:         "audio/sfx/death.ogg",
	SFXVictory:       "audio/sfx/victory.ogg",
	SFXDefeat:        "audio/sfx/defeat.ogg",
	SFXEncounter:     "audio/sfx/encounter.ogg",
	SFXCardDraw:      "audio/sfx/card_draw.ogg",
	SFXLandPlay:      "audio/sfx/land_play.ogg",
	SFXCounter:       "audio/sfx/counter.ogg",
	SFXCreatureDeath: "audio/sfx/creature_death.ogg",

	SFXEnemyDragon:   "audio/sfx/enemy_dragon.ogg",
	SFXEnemyKnight:   "audio/sfx/enemy_knight.ogg",
	SFXEnemyWolf:     "audio/sfx/enemy_wolf.ogg",
	SFXEnemyTroll:    "audio/sfx/enemy_troll.ogg",
	SFXEnemyDjinn:    "audio/sfx/enemy_djinn.ogg",
	SFXEnemyWyrm:     "audio/sfx/enemy_wyrm.ogg",
	SFXEnemyArchmage: "audio/sfx/enemy_archmage.ogg",
	SFXEnemyLord:     "audio/sfx/enemy_lord.ogg",
	SFXEnemyHorse:    "audio/sfx/enemy_horse.ogg",
	SFXEnemyFlying:   "audio/sfx/enemy_flying.ogg",

	SFXWizardFemale: "audio/sfx/wizard_female.ogg",
	SFXWizardMale:   "audio/sfx/wizard_male.ogg",

	SFXFindCard:  "audio/sfx/findcard.ogg",
	SFXTreasure:  "audio/sfx/treasure.ogg",
	SFXReward:    "audio/sfx/reward.ogg",
	SFXManaball:  "audio/sfx/manaball.ogg",
	SFXScroll:    "audio/sfx/scroll.ogg",
	SFXNewsflash: "audio/sfx/newsflash.ogg",
	SFXManalink:  "audio/sfx/manalink.ogg",
	SFXDice:      "audio/sfx/dice.ogg",
	SFXSummon:    "audio/sfx/summon.ogg",

	SFXWinGame:     "audio/sfx/wingame.ogg",
	SFXStatsScreen: "audio/sfx/statsscreen.ogg",

	SFXWorldMagicWhite: "audio/sfx/worldmagic_white.ogg",
	SFXWorldMagicBlue:  "audio/sfx/worldmagic_blue.ogg",
	SFXWorldMagicBlack: "audio/sfx/worldmagic_black.ogg",
	SFXWorldMagicRed:   "audio/sfx/worldmagic_red.ogg",
	SFXWorldMagicGreen: "audio/sfx/worldmagic_green.ogg",

	SFXCastleDefault: "audio/sfx/castle_default.ogg",
	SFXCastleBlue:    "audio/sfx/castle_blue.ogg",
	SFXCastleBlack:   "audio/sfx/castle_black.ogg",
	SFXCastleRed:     "audio/sfx/castle_red.ogg",
	SFXCastleGreen:   "audio/sfx/castle_green.ogg",
}

func (s SFX) String() string {
	for sfx, path := range sfxFiles {
		if sfx == s {
			return path
		}
	}
	return fmt.Sprintf("sfx_%d", s)
}

// TerrainColor maps terrain types to a mana color for sound selection.
type TerrainColor int

const (
	TerrainColorWhite TerrainColor = iota // Plains, Sand
	TerrainColorBlue                      // Water, Island
	TerrainColorBlack                     // Marsh, Swamp
	TerrainColorRed                       // Mountains, Snow
	TerrainColorGreen                     // Forest
)

// TerrainTypeToColor maps world terrain type constants to audio terrain colors.
// Terrain type constants from world/generate.go:
//
//	0=Undefined, 1=Water, 2=Sand, 3=Marsh, 4=Plains, 5=Forest, 6=Mountains, 7=Snow
func TerrainTypeToColor(terrainType int) TerrainColor {
	switch terrainType {
	case 1: // Water
		return TerrainColorBlue
	case 2: // Sand
		return TerrainColorWhite
	case 3: // Marsh
		return TerrainColorBlack
	case 4: // Plains
		return TerrainColorWhite
	case 5: // Forest
		return TerrainColorGreen
	case 6: // Mountains
		return TerrainColorRed
	case 7: // Snow
		return TerrainColorRed
	default:
		return TerrainColorWhite
	}
}

// Footstep sound file pairs (left, right) per terrain color.
var footstepFiles = map[TerrainColor][2]string{
	TerrainColorWhite: {"audio/sfx/walk_white_l.ogg", "audio/sfx/walk_white_r.ogg"},
	TerrainColorBlue:  {"audio/sfx/walk_blue_l.ogg", "audio/sfx/walk_blue_r.ogg"},
	TerrainColorBlack: {"audio/sfx/walk_black_l.ogg", "audio/sfx/walk_black_r.ogg"},
	TerrainColorRed:   {"audio/sfx/walk_red_l.ogg", "audio/sfx/walk_red_r.ogg"},
	TerrainColorGreen: {"audio/sfx/walk_green_l.ogg", "audio/sfx/walk_green_r.ogg"},
}

// Bird ambient sound files per terrain color (multiple variants).
var birdFiles = map[TerrainColor][]string{
	TerrainColorWhite: {"audio/sfx/bird_white_1.ogg"},
	TerrainColorBlue:  {"audio/sfx/bird_blue_1.ogg"},
	TerrainColorBlack: {"audio/sfx/bird_black_1.ogg", "audio/sfx/bird_black_2.ogg"},
	TerrainColorRed:   {"audio/sfx/bird_red_1.ogg", "audio/sfx/bird_red_2.ogg"},
	TerrainColorGreen: {"audio/sfx/bird_green_1.ogg"},
}

// BGM identifies a background music track.
type BGM int

const (
	BGMBattle BGM = iota
	BGMCity
	BGMTitle
	BGMDungeon
	BGMTemple
	// World tracks (Locmus0-19)
	BGMWorld0
	BGMWorld1
	BGMWorld2
	BGMWorld3
	BGMWorld4
	BGMWorld5
	BGMWorld6
	BGMWorld7
	BGMWorld8
	BGMWorld9
	BGMWorld10
	BGMWorld11
	BGMWorld12
	BGMWorld13
	BGMWorld14
	BGMWorld15
	BGMWorld16
	BGMWorld17
	BGMWorld18
	BGMWorld19

	BGMNone BGM = -1
)

var bgmFiles = map[BGM]string{
	BGMBattle:  "audio/bgm/battle.ogg",
	BGMCity:    "audio/bgm/city.ogg",
	BGMTitle:   "audio/bgm/title.ogg",
	BGMDungeon: "audio/bgm/dungeon.ogg",
	BGMTemple:  "audio/bgm/temple.ogg",
	BGMWorld0:  "audio/bgm/world_0.ogg",
	BGMWorld1:  "audio/bgm/world_1.ogg",
	BGMWorld2:  "audio/bgm/world_2.ogg",
	BGMWorld3:  "audio/bgm/world_3.ogg",
	BGMWorld4:  "audio/bgm/world_4.ogg",
	BGMWorld5:  "audio/bgm/world_5.ogg",
	BGMWorld6:  "audio/bgm/world_6.ogg",
	BGMWorld7:  "audio/bgm/world_7.ogg",
	BGMWorld8:  "audio/bgm/world_8.ogg",
	BGMWorld9:  "audio/bgm/world_9.ogg",
	BGMWorld10: "audio/bgm/world_10.ogg",
	BGMWorld11: "audio/bgm/world_11.ogg",
	BGMWorld12: "audio/bgm/world_12.ogg",
	BGMWorld13: "audio/bgm/world_13.ogg",
	BGMWorld14: "audio/bgm/world_14.ogg",
	BGMWorld15: "audio/bgm/world_15.ogg",
	BGMWorld16: "audio/bgm/world_16.ogg",
	BGMWorld17: "audio/bgm/world_17.ogg",
	BGMWorld18: "audio/bgm/world_18.ogg",
	BGMWorld19: "audio/bgm/world_19.ogg",
}

var worldBGMs = []BGM{
	BGMWorld0, BGMWorld1, BGMWorld2, BGMWorld3, BGMWorld4,
	BGMWorld5, BGMWorld6, BGMWorld7, BGMWorld8, BGMWorld9,
	BGMWorld10, BGMWorld11, BGMWorld12, BGMWorld13, BGMWorld14,
	BGMWorld15, BGMWorld16, BGMWorld17, BGMWorld18, BGMWorld19,
}

func (b BGM) String() string {
	if path, ok := bgmFiles[b]; ok {
		return path
	}
	return fmt.Sprintf("bgm_%d", b)
}

// AudioManager handles all game audio playback.
type AudioManager struct {
	context       *audio.Context
	bgmPlayer     *audio.Player
	currentBGM    BGM
	sfxBytes      map[SFX][]byte
	footstepBytes map[TerrainColor][2][]byte // [0]=left, [1]=right
	birdBytes     map[TerrainColor][][]byte
	bgmVolume     float64
	sfxVolume     float64
	muted         bool
	footstepLeft  bool // alternates L/R
}

var instance *AudioManager

// Get returns the global AudioManager instance. Returns nil if not yet initialized.
func Get() *AudioManager {
	return instance
}

// NewAudioManager creates a new AudioManager with default volumes and sets it as the global instance.
func NewAudioManager() *AudioManager {
	am := &AudioManager{
		currentBGM:    BGMNone,
		sfxBytes:      make(map[SFX][]byte),
		footstepBytes: make(map[TerrainColor][2][]byte),
		birdBytes:     make(map[TerrainColor][][]byte),
		bgmVolume:     0.4,
		sfxVolume:     0.7,
	}

	if ctx := audio.CurrentContext(); ctx != nil {
		am.context = ctx
	} else {
		am.context = audio.NewContext(sampleRate)
	}
	am.preloadSFX()
	am.preloadFootsteps()
	am.preloadBirds()

	instance = am
	return am
}

func (am *AudioManager) preloadSFX() {
	for sfx, path := range sfxFiles {
		am.sfxBytes[sfx] = decodeOgg(path)
	}
}

func (am *AudioManager) preloadFootsteps() {
	for color, pair := range footstepFiles {
		am.footstepBytes[color] = [2][]byte{
			decodeOgg(pair[0]),
			decodeOgg(pair[1]),
		}
	}
}

func (am *AudioManager) preloadBirds() {
	for color, files := range birdFiles {
		var decoded [][]byte
		for _, path := range files {
			if d := decodeOgg(path); d != nil {
				decoded = append(decoded, d)
			}
		}
		am.birdBytes[color] = decoded
	}
}

func decodeOgg(path string) []byte {
	data, err := assets.AudioFS.ReadFile(path)
	if err != nil {
		fmt.Printf("audio: failed to load %s: %v\n", path, err)
		return nil
	}

	stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("audio: failed to decode %s: %v\n", path, err)
		return nil
	}

	decoded, err := io.ReadAll(stream)
	if err != nil {
		fmt.Printf("audio: failed to read decoded %s: %v\n", path, err)
		return nil
	}

	return decoded
}

// PlaySFX plays a sound effect. Fire-and-forget.
func (am *AudioManager) PlaySFX(sfx SFX) {
	if am.muted || am.context == nil {
		return
	}

	data, ok := am.sfxBytes[sfx]
	if !ok || data == nil {
		return
	}

	player := am.context.NewPlayerFromBytes(data)
	player.SetVolume(am.sfxVolume)
	player.Play()
}

// PlayFootstep plays a terrain-colored footstep sound, alternating left/right.
func (am *AudioManager) PlayFootstep(color TerrainColor) {
	if am.muted || am.context == nil {
		return
	}

	pair, ok := am.footstepBytes[color]
	if !ok {
		return
	}

	idx := 0
	if am.footstepLeft {
		idx = 1
	}
	am.footstepLeft = !am.footstepLeft

	data := pair[idx]
	if data == nil {
		return
	}

	player := am.context.NewPlayerFromBytes(data)
	player.SetVolume(am.sfxVolume * 0.5)
	player.Play()
}

// PlayBird plays a random bird ambient sound for the given terrain color.
func (am *AudioManager) PlayBird(color TerrainColor) {
	if am.muted || am.context == nil {
		return
	}

	birds, ok := am.birdBytes[color]
	if !ok || len(birds) == 0 {
		return
	}

	data := birds[rand.Intn(len(birds))]
	if data == nil {
		return
	}

	player := am.context.NewPlayerFromBytes(data)
	player.SetVolume(am.sfxVolume * 0.3)
	player.Play()
}

// RandomWorldBGM returns a random world exploration BGM track.
func RandomWorldBGM() BGM {
	return worldBGMs[rand.Intn(len(worldBGMs))]
}

// IsWorldBGM returns true if the given BGM is one of the world exploration tracks.
func IsWorldBGM(bgm BGM) bool {
	return bgm >= BGMWorld0 && bgm <= BGMWorld19
}

// PlayBGM switches to a new background music track (looping).
func (am *AudioManager) PlayBGM(bgm BGM) {
	if bgm == am.currentBGM {
		return
	}

	am.StopBGM()
	am.currentBGM = bgm

	if am.muted || am.context == nil {
		return
	}

	am.startBGM(bgm)
}

func (am *AudioManager) startBGM(bgm BGM) {
	path, ok := bgmFiles[bgm]
	if !ok {
		return
	}

	data, err := assets.AudioFS.ReadFile(path)
	if err != nil {
		fmt.Printf("audio: failed to load BGM %s: %v\n", path, err)
		return
	}

	stream, err := vorbis.DecodeWithSampleRate(sampleRate, bytes.NewReader(data))
	if err != nil {
		fmt.Printf("audio: failed to decode BGM %s: %v\n", path, err)
		return
	}

	loop := audio.NewInfiniteLoop(stream, stream.Length())

	player, err := am.context.NewPlayer(loop)
	if err != nil {
		fmt.Printf("audio: failed to create BGM player: %v\n", err)
		return
	}

	player.SetVolume(am.bgmVolume)
	player.Play()
	am.bgmPlayer = player
}

// CurrentBGM returns the currently playing BGM track.
func (am *AudioManager) CurrentBGM() BGM {
	return am.currentBGM
}

// StopBGM stops the current background music.
func (am *AudioManager) StopBGM() {
	if am.bgmPlayer != nil {
		am.bgmPlayer.Close()
		am.bgmPlayer = nil
	}
	am.currentBGM = BGMNone
}

// SetVolume sets BGM and SFX volumes (0.0 to 1.0).
func (am *AudioManager) SetVolume(bgm, sfx float64) {
	am.bgmVolume = clamp(bgm, 0, 1)
	am.sfxVolume = clamp(sfx, 0, 1)

	if am.bgmPlayer != nil {
		am.bgmPlayer.SetVolume(am.bgmVolume)
	}
}

// Mute mutes all audio.
func (am *AudioManager) Mute() {
	am.muted = true
	if am.bgmPlayer != nil {
		am.bgmPlayer.Pause()
	}
}

// Unmute restores audio playback.
func (am *AudioManager) Unmute() {
	am.muted = false
	if am.bgmPlayer != nil {
		am.bgmPlayer.Play()
	} else if am.currentBGM != BGMNone && am.context != nil {
		am.startBGM(am.currentBGM)
	}
}

// ToggleMute toggles the mute state.
func (am *AudioManager) ToggleMute() {
	if am.muted {
		am.Unmute()
	} else {
		am.Mute()
	}
}

// EnemySFXForName returns the appropriate SFX for an enemy based on its name.
func EnemySFXForName(name string) SFX {
	keywords := map[string]SFX{
		"Dragon":   SFXEnemyDragon,
		"Drake":    SFXEnemyDragon,
		"Hydra":    SFXEnemyDragon,
		"Dracur":   SFXEnemyDragon,
		"Knight":   SFXEnemyKnight,
		"Crusader": SFXEnemyKnight,
		"Paladin":  SFXEnemyKnight,
		"Wolf":     SFXEnemyWolf,
		"Beast":    SFXEnemyWolf,
		"Sedge":    SFXEnemyWolf,
		"Troll":    SFXEnemyTroll,
		"Fungus":   SFXEnemyTroll,
		"Nether":   SFXEnemyTroll,
		"Djinn":    SFXEnemyDjinn,
		"Prismat":  SFXEnemyDjinn,
		"Wyrm":     SFXEnemyWyrm,
		"Archmage": SFXEnemyArchmage,
		"Arzakon":  SFXEnemyArchmage,
		"Lord":     SFXEnemyLord,
		"Vampire":  SFXEnemyLord,
		"Lich":     SFXEnemyLord,
		"Stallion": SFXEnemyHorse,
		"Centaur":  SFXEnemyHorse,
		"Winged":   SFXEnemyFlying,
		"Angel":    SFXEnemyFlying,
		"Astral":   SFXEnemyFlying,
	}

	for keyword, sfx := range keywords {
		for range name {
			if strings.Contains(name, keyword) {
				return sfx
			}
		}
	}

	return SFXEncounter
}

// CastleSFXForColor returns the castle entry SFX for a given color string.
func CastleSFXForColor(colorStr string) SFX {
	switch colorStr {
	case "Blue":
		return SFXCastleBlue
	case "Black":
		return SFXCastleBlack
	case "Red":
		return SFXCastleRed
	case "Green":
		return SFXCastleGreen
	default:
		return SFXCastleDefault
	}
}

// WorldMagicSFXForColor returns the world magic SFX for a given color string.
func WorldMagicSFXForColor(colorStr string) SFX {
	switch colorStr {
	case "White":
		return SFXWorldMagicWhite
	case "Blue":
		return SFXWorldMagicBlue
	case "Black":
		return SFXWorldMagicBlack
	case "Red":
		return SFXWorldMagicRed
	case "Green":
		return SFXWorldMagicGreen
	default:
		return SFXWorldMagicWhite
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
