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
		ambientCache:  make(map[string][]byte),
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
		SFXLandPlay:      "audio/sfx/land_play.ogg",
		SFXManaball:      "audio/sfx/manaball.ogg",
		SFXManalink:      "audio/sfx/manalink.ogg",
		SFXReward:        "audio/sfx/reward.ogg",
		SFXSummon:        "audio/sfx/summon.ogg",
	}
	for sfx, want := range wantSFX {
		if got := sfxFiles[sfx]; got != want {
			t.Errorf("sfxFiles[%v] = %q, want %q", sfx, got, want)
		}
	}

	wantBGM := map[BGM]string{
		BGMCastleDefault: "audio/bgm/castle_default.ogg",
		BGMCastleBlue:    "audio/bgm/castle_blue.ogg",
		BGMCastleBlack:   "audio/bgm/castle_black.ogg",
		BGMCastleRed:     "audio/bgm/castle_red.ogg",
		BGMCastleGreen:   "audio/bgm/castle_green.ogg",
		BGMDeath:         "audio/bgm/death.ogg",
		BGMDefeat:        "audio/bgm/defeat.ogg",
		BGMStatsScreen:   "audio/bgm/statsscreen.ogg",
		BGMVictory:       "audio/bgm/victory.ogg",
		BGMWinGame:       "audio/bgm/wingame.ogg",
	}
	for bgm, want := range wantBGM {
		if got := bgmFiles[bgm]; got != want {
			t.Errorf("bgmFiles[%v] = %q, want %q", bgm, got, want)
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

func TestTargetSampleRate(t *testing.T) {
	tests := []struct {
		name string
		web  bool
		want int
	}{
		{name: "native", want: 22050},
		{name: "web", web: true, want: 48000},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := targetSampleRate(test.web); got != test.want {
				t.Fatalf("targetSampleRate(%t) = %d, want %d", test.web, got, test.want)
			}
		})
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

func TestPlayBGMReplacesCurrentTrack(t *testing.T) {
	am := newTestAudioManager()
	am.Mute()
	am.PlayBGM(BGMVictory)
	am.PlayBGM(BGMDefeat)

	if got := am.CurrentBGM(); got != BGMDefeat {
		t.Fatalf("current BGM = %v, want %v", got, BGMDefeat)
	}
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

func TestIsCastleBGM(t *testing.T) {
	for _, bgm := range []BGM{
		BGMCastleDefault,
		BGMCastleBlue,
		BGMCastleBlack,
		BGMCastleRed,
		BGMCastleGreen,
	} {
		if !IsCastleBGM(bgm) {
			t.Errorf("%v should be a castle BGM", bgm)
		}
	}
	if IsCastleBGM(BGMCity) {
		t.Error("BGMCity should not be a castle BGM")
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

func TestCastleBGMForColor(t *testing.T) {
	if CastleBGMForColor("Blue") != BGMCastleBlue {
		t.Error("expected BGMCastleBlue for Blue")
	}
	if CastleBGMForColor("Black") != BGMCastleBlack {
		t.Error("expected BGMCastleBlack for Black")
	}
	if CastleBGMForColor("Red") != BGMCastleRed {
		t.Error("expected BGMCastleRed for Red")
	}
	if CastleBGMForColor("Green") != BGMCastleGreen {
		t.Error("expected BGMCastleGreen for Green")
	}
	if CastleBGMForColor("White") != BGMCastleDefault {
		t.Error("expected BGMCastleDefault for White")
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

func TestLazySFXDecodingCachesDecodedAudio(t *testing.T) {
	am := newTestAudioManager()
	if len(am.sfxBytes) != 0 {
		t.Fatalf("expected sfxBytes to start empty, got %d entries", len(am.sfxBytes))
	}

	data := am.getOrDecodeSFX(SFXClick)
	if len(data) == 0 {
		t.Fatal("expected decoded SFXClick audio data")
	}
	if cached := am.sfxBytes[SFXClick]; len(cached) == 0 {
		t.Fatal("expected SFXClick to be cached in sfxBytes")
	}

	second := am.getOrDecodeSFX(SFXClick)
	if &data[0] != &second[0] {
		t.Fatal("expected subsequent getOrDecodeSFX call to return cached slice")
	}
}

func TestLazyFootstepDecodingCachesDecodedAudio(t *testing.T) {
	am := newTestAudioManager()
	data := am.getOrDecodeFootstep(TerrainColorWhite, 0)
	if len(data) == 0 {
		t.Fatal("expected decoded footstep audio data")
	}
	if cached := am.footstepBytes[TerrainColorWhite][0]; len(cached) == 0 {
		t.Fatal("expected footstep to be cached in footstepBytes")
	}
}

func TestLazyAmbientDecodingCachesDecodedAudio(t *testing.T) {
	am := newTestAudioManager()
	path := dungeonFiles[0]
	data := am.getOrDecodeAmbient(path)
	if len(data) == 0 {
		t.Fatal("expected decoded ambient audio data")
	}
	if cached := am.ambientCache[path]; len(cached) == 0 {
		t.Fatal("expected ambient data to be cached in ambientCache")
	}
}

