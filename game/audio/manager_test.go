package audio

import (
	"slices"
	"testing"
)

type fakeAmbientPlayer struct {
	playing   bool
	playCalls int
	volume    float64
}

func (p *fakeAmbientPlayer) IsPlaying() bool {
	return p.playing
}

func (p *fakeAmbientPlayer) Play() {
	p.playCalls++
	p.playing = true
}

func (p *fakeAmbientPlayer) SetVolume(volume float64) {
	p.volume = volume
}

// newTestAudioManager creates an AudioManager without preloading audio files.
func newTestAudioManager() *AudioManager {
	am := &AudioManager{
		currentBGM:    BGMNone,
		sfxBytes:      make(map[SFX][]byte),
		footstepBytes: make(map[TerrainColor][2][]byte),
		birdBytes:     make(map[TerrainColor][][]byte),
		landBytes:     make(map[TerrainColor][][]byte),
		dungeonBytes:  make([][]byte, 0),
		bgmVolume:     0.2,
		sfxVolume:     0.7,
	}
	instance = am
	return am
}

func TestRequestedSoundAssetsAreMapped(t *testing.T) {
	wantSFX := map[SFX]string{
		SFXClick:         "audio/sfx/click.ogg",
		SFXClick2:        "audio/sfx/click2.ogg",
		SFXCast:          "audio/sfx/cast.ogg",
		SFXCounter:       "audio/sfx/counter.ogg",
		SFXCreatureDeath: "audio/sfx/creature_death.ogg",
		SFXDamage:        "audio/sfx/damage.ogg",
		SFXDefeat:        "audio/sfx/defeat.ogg",
		SFXLandPlay:      "audio/sfx/land_play.ogg",
		SFXManaball:      "audio/sfx/manaball.ogg",
		SFXManalink:      "audio/sfx/manalink.ogg",
		SFXReward:        "audio/sfx/reward.ogg",
		SFXSummon:        "audio/sfx/summon.ogg",
		SFXWinGame:       "audio/sfx/wingame.ogg",
		SFXDeath:         "audio/sfx/death.ogg",
	}
	for sfx, want := range wantSFX {
		if got := sfxFiles[sfx]; got != want {
			t.Errorf("sfxFiles[%v] = %q, want %q", sfx, got, want)
		}
	}

	for _, color := range []TerrainColor{
		TerrainColorWhite, TerrainColorBlue, TerrainColorBlack, TerrainColorRed, TerrainColorGreen,
	} {
		if len(footstepFiles[color]) != 2 {
			t.Errorf("terrain %v has no footstep pair", color)
		}
		if len(birdFiles[color]) == 0 {
			t.Errorf("terrain %v has no bird ambience", color)
		}
		if len(landFiles[color]) == 0 {
			t.Errorf("terrain %v has no land ambience", color)
		}
	}
	if len(dungeonFiles) != 5 {
		t.Fatalf("len(dungeonFiles) = %d, want 5", len(dungeonFiles))
	}
}

func TestTrimPCM(t *testing.T) {
	data := make([]byte, sampleRate*bytesPerSampleFrame*2)
	got := trimPCM(data, 500, 1000)
	wantLen := sampleRate * bytesPerSampleFrame / 2
	if len(got) != wantLen {
		t.Fatalf("len(trimPCM()) = %d, want %d", len(got), wantLen)
	}
	if !slices.Equal(got, data[wantLen:wantLen*2]) {
		t.Fatal("trimPCM() returned the wrong half-second window")
	}
}

func TestNewAudioManager(t *testing.T) {
	am := newTestAudioManager()
	if am.bgmVolume != 0.2 {
		t.Errorf("expected bgmVolume=0.2, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 0.7 {
		t.Errorf("expected sfxVolume=0.7, got %f", am.sfxVolume)
	}
	if am.muted {
		t.Error("expected muted=false")
	}
}

func TestMuteUnmute(t *testing.T) {
	am := newTestAudioManager()

	am.Mute()
	if !am.muted {
		t.Error("expected muted after Mute()")
	}

	am.Unmute()
	if am.muted {
		t.Error("expected unmuted after Unmute()")
	}
}

func TestToggleMute(t *testing.T) {
	am := newTestAudioManager()

	am.ToggleMute()
	if !am.muted {
		t.Error("expected muted after first toggle")
	}

	am.ToggleMute()
	if am.muted {
		t.Error("expected unmuted after second toggle")
	}
}

func TestSetVolume(t *testing.T) {
	am := newTestAudioManager()

	am.SetVolume(0.5, 0.8)
	if am.bgmVolume != 0.5 {
		t.Errorf("expected bgmVolume=0.5, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 0.8 {
		t.Errorf("expected sfxVolume=0.8, got %f", am.sfxVolume)
	}
}

func TestSetVolumeClamped(t *testing.T) {
	am := newTestAudioManager()

	am.SetVolume(-0.5, 1.5)
	if am.bgmVolume != 0.0 {
		t.Errorf("expected bgmVolume=0.0, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 1.0 {
		t.Errorf("expected sfxVolume=1.0, got %f", am.sfxVolume)
	}
}

func TestPlaySFXWhenMuted(t *testing.T) {
	am := newTestAudioManager()
	am.Mute()
	am.PlaySFX(SFXClick)
}

func TestPlayBGMWhenMuted(t *testing.T) {
	am := newTestAudioManager()
	am.Mute()
	am.PlayBGM(BGMWorld0)
}

func TestStopBGM(t *testing.T) {
	am := newTestAudioManager()
	am.StopBGM()
}

func TestRandomWorldBGM(t *testing.T) {
	bgm := RandomWorldBGM()
	if !IsWorldBGM(bgm) {
		t.Errorf("RandomWorldBGM returned non-world BGM: %d", bgm)
	}
}

func TestRandomCityBGMUsesTierRanges(t *testing.T) {
	tests := []struct {
		tier     int
		min, max BGM
	}{
		{tier: 1, min: BGMWorld0, max: BGMWorld6},
		{tier: 2, min: BGMWorld7, max: BGMWorld13},
		{tier: 3, min: BGMWorld14, max: BGMWorld19},
	}
	for _, test := range tests {
		for range 100 {
			got := RandomCityBGM(test.tier)
			if got < test.min || got > test.max {
				t.Fatalf("RandomCityBGM(%d) = %v, want range %v..%v", test.tier, got, test.min, test.max)
			}
		}
	}
}

func TestIsWorldBGM(t *testing.T) {
	if !IsWorldBGM(BGMWorld0) {
		t.Error("BGMWorld0 should be a world BGM")
	}
	if !IsWorldBGM(BGMWorld19) {
		t.Error("BGMWorld19 should be a world BGM")
	}
	if IsWorldBGM(BGMBattle) {
		t.Error("BGMBattle should not be a world BGM")
	}
	if IsWorldBGM(BGMCity) {
		t.Error("BGMCity should not be a world BGM")
	}
}

func TestEnemySFXForName(t *testing.T) {
	tests := []struct {
		name     string
		expected SFX
	}{
		{"Forest Dragon", SFXEnemyDragon},
		{"Sea Drake", SFXEnemyDragon},
		{"Crag Hydra", SFXEnemyDragon},
		{"Undead Knight", SFXEnemyKnight},
		{"Crusader", SFXEnemyKnight},
		{"Paladin", SFXEnemyKnight},
		{"Sedge Beast", SFXEnemyWolf},
		{"Beast Master", SFXEnemyWolf},
		{"Troll Shaman", SFXEnemyTroll},
		{"Nether Fiend", SFXEnemyTroll},
		{"Prismat", SFXEnemyDjinn},
		{"Arzakon", SFXEnemyArchmage},
		{"Vampire Lord", SFXEnemyLord},
		{"Winged Stallion", SFXEnemyHorse},
		{"Winged Serpent", SFXEnemyFlying},
		{"Arch Angel", SFXEnemyFlying},
		{"Goblin Warlord", SFXEncounter},
		{"Sorcerer", SFXEncounter},
	}

	for _, tt := range tests {
		got := EnemySFXForName(tt.name)
		if got != tt.expected {
			t.Errorf("EnemySFXForName(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}

func TestCastleSFXForColor(t *testing.T) {
	if CastleSFXForColor("Blue") != SFXCastleBlue {
		t.Error("expected SFXCastleBlue for Blue")
	}
	if CastleSFXForColor("Black") != SFXCastleBlack {
		t.Error("expected SFXCastleBlack for Black")
	}
	if CastleSFXForColor("Red") != SFXCastleRed {
		t.Error("expected SFXCastleRed for Red")
	}
	if CastleSFXForColor("Green") != SFXCastleGreen {
		t.Error("expected SFXCastleGreen for Green")
	}
	if CastleSFXForColor("White") != SFXCastleDefault {
		t.Error("expected SFXCastleDefault for White")
	}
}

func TestWorldMagicSFXForColor(t *testing.T) {
	if WorldMagicSFXForColor("White") != SFXWorldMagicWhite {
		t.Error("expected SFXWorldMagicWhite")
	}
	if WorldMagicSFXForColor("Blue") != SFXWorldMagicBlue {
		t.Error("expected SFXWorldMagicBlue")
	}
}

func TestPlayFootstepWhenMuted(t *testing.T) {
	am := newTestAudioManager()
	am.Mute()
	am.PlayFootstep(TerrainColorWhite)
}

func TestPlayBirdWhenMuted(t *testing.T) {
	am := newTestAudioManager()
	am.Mute()
	am.PlayBird(TerrainColorGreen)
}

func TestAmbientSoundsDoNotOverlap(t *testing.T) {
	am := newTestAudioManager()
	am.birdBytes[TerrainColorGreen] = [][]byte{{1}}
	am.landBytes[TerrainColorGreen] = [][]byte{{2}}
	am.dungeonBytes = [][]byte{{3}}

	var players []*fakeAmbientPlayer
	am.newAmbientPlayer = func([]byte) ambientPlayer {
		player := &fakeAmbientPlayer{}
		players = append(players, player)
		return player
	}

	am.PlayBird(TerrainColorGreen)
	am.PlayLandAmbience(TerrainColorGreen)
	am.PlayDungeonAmbience()

	if len(players) != 1 {
		t.Fatalf("created %d ambient players while the first was active, want 1", len(players))
	}
	if players[0].playCalls != 1 {
		t.Fatalf("first ambient player Play calls = %d, want 1", players[0].playCalls)
	}
	if want := am.sfxVolume * 0.3; players[0].volume != want {
		t.Fatalf("first ambient player volume = %f, want %f", players[0].volume, want)
	}

	players[0].playing = false
	am.PlayDungeonAmbience()
	if len(players) != 2 {
		t.Fatalf("created %d ambient players after the first finished, want 2", len(players))
	}
}

func TestGetInstance(t *testing.T) {
	am := newTestAudioManager()
	if Get() != am {
		t.Error("Get() should return the instance set by NewAudioManager")
	}
}
