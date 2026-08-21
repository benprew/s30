package bugreport

import (
	"fmt"
	"runtime"
	"time"

	"github.com/benprew/s30/game/save"
	"github.com/benprew/s30/game/world"
	"github.com/hajimehoshi/ebiten/v2"
)

const GameVersion = "1.0.0"

// DuelReporter is implemented by DuelScreen to supply structured duel report data.
type DuelReporter interface {
	DuelReportState() *DuelReportState
}

// CollectEnvironment captures current hardware and runtime statistics.
func CollectEnvironment(activeScreen string) EnvironmentInfo {
	var fps, tps float64
	// In tests, ebiten graphics context may not be running.
	defer func() {
		_ = recover()
	}()
	fps = ebiten.ActualFPS()
	tps = ebiten.ActualTPS()

	return EnvironmentInfo{
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		GoVersion:    runtime.Version(),
		GameVersion:  GameVersion,
		ActiveScreen: activeScreen,
		FPS:          fps,
		TPS:          tps,
		WindowWidth:  1024,
		WindowHeight: 768,
	}
}

// CollectReport builds a BugReport from current game state.
func CollectReport(level *world.Level, activeScreenName string, activeScreen any, userNotes string) *BugReport {
	now := time.Now().UTC()
	id := fmt.Sprintf("bug_%s_%d", now.Format("2006-01-02_150405"), now.Nanosecond()/1e6)

	report := &BugReport{
		ID:           id,
		Timestamp:    now,
		UserNotes:    userNotes,
		Environment:  CollectEnvironment(activeScreenName),
		ActiveScreen: activeScreenName,
	}

	if level != nil {
		report.WorldState = &save.SaveData{
			Name:    level.SaveName(),
			GameID:  level.GameID,
			Version: 1,
			SavedAt: now,
			World:   level,
		}
	}

	if reporter, ok := activeScreen.(DuelReporter); ok && reporter != nil {
		report.DuelState = reporter.DuelReportState()
	}

	return report
}

// CollectCrashReport captures an unexpected panic and runtime stack trace.
func CollectCrashReport(level *world.Level, activeScreenName string, activeScreen any, panicVal any, stack []byte) *BugReport {
	now := time.Now().UTC()
	id := fmt.Sprintf("crash_%s_%d", now.Format("2006-01-02_150405"), now.Nanosecond()/1e6)

	var crashMsg string
	if panicVal != nil {
		crashMsg = fmt.Sprintf("%v", panicVal)
	} else {
		crashMsg = "unknown panic"
	}

	report := &BugReport{
		ID:           id,
		Timestamp:    now,
		IsCrash:      true,
		CrashMessage: crashMsg,
		StackTrace:   string(stack),
		Environment:  CollectEnvironment(activeScreenName),
		ActiveScreen: activeScreenName,
	}

	if level != nil {
		report.WorldState = &save.SaveData{
			Name:    level.SaveName(),
			GameID:  level.GameID,
			Version: 1,
			SavedAt: now,
			World:   level,
		}
	}

	if reporter, ok := activeScreen.(DuelReporter); ok && reporter != nil {
		report.DuelState = reporter.DuelReportState()
	}

	return report
}
