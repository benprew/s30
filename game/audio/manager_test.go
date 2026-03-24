package audio

import (
	"testing"
)

func TestNewAudioManager(t *testing.T) {
	am := NewAudioManager()
	if am == nil {
		t.Fatal("expected non-nil AudioManager")
	}
	if am.bgmVolume != 0.4 {
		t.Errorf("expected bgmVolume=0.4, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 0.7 {
		t.Errorf("expected sfxVolume=0.7, got %f", am.sfxVolume)
	}
	if am.muted {
		t.Error("expected muted=false")
	}
}

func TestMuteUnmute(t *testing.T) {
	am := NewAudioManager()

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
	am := NewAudioManager()

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
	am := NewAudioManager()

	am.SetVolume(0.5, 0.8)
	if am.bgmVolume != 0.5 {
		t.Errorf("expected bgmVolume=0.5, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 0.8 {
		t.Errorf("expected sfxVolume=0.8, got %f", am.sfxVolume)
	}
}

func TestSetVolumeClamped(t *testing.T) {
	am := NewAudioManager()

	am.SetVolume(-0.5, 1.5)
	if am.bgmVolume != 0.0 {
		t.Errorf("expected bgmVolume=0.0, got %f", am.bgmVolume)
	}
	if am.sfxVolume != 1.0 {
		t.Errorf("expected sfxVolume=1.0, got %f", am.sfxVolume)
	}
}

func TestPlaySFXWhenMuted(t *testing.T) {
	am := NewAudioManager()
	am.Mute()
	am.PlaySFX(SFXClick)
}

func TestPlayBGMWhenMuted(t *testing.T) {
	am := NewAudioManager()
	am.Mute()
	am.PlayBGM(BGMWorld0)
}

func TestStopBGM(t *testing.T) {
	am := NewAudioManager()
	am.StopBGM()
}

func TestRandomWorldBGM(t *testing.T) {
	bgm := RandomWorldBGM()
	if !IsWorldBGM(bgm) {
		t.Errorf("RandomWorldBGM returned non-world BGM: %d", bgm)
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
		{"Winged Stallion", SFXEnemyFlying},
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
	am := NewAudioManager()
	am.Mute()
	am.PlayFootstep(TerrainColorWhite)
}

func TestPlayBirdWhenMuted(t *testing.T) {
	am := NewAudioManager()
	am.Mute()
	am.PlayBird(TerrainColorGreen)
}

func TestGetInstance(t *testing.T) {
	am := NewAudioManager()
	if Get() != am {
		t.Error("Get() should return the instance set by NewAudioManager")
	}
}
