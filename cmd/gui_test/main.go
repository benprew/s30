// Package main provides automated headless GUI testing for s30 using Ebitengine 2.10's
// Application Virtualization (VM host) mode. It drives guests headlessly without opening any window
// or requiring an X server / Xvfb.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

type scenarioConfig struct {
	name     string
	pkg      string
	width    int
	height   int
	runTest  func(d *guiDriver) error
}

type guiDriver struct {
	guest              *vmhost.GuestSession
	screen             *ebiten.Image
	outDir             string
	scenarioName       string
	snapshots          []string
	runScript          func(d *guiDriver) error
	testErr            error
	guestSessionClosed bool
	closeErr           error
}

func (d *guiDriver) Update() error {
	d.screen = ebiten.NewImage(d.screen.Bounds().Dx(), d.screen.Bounds().Dy())
	if err := d.guest.SetOutsideScreen(d.screen); err != nil {
		d.testErr = fmt.Errorf("SetOutsideScreen: %w", err)
		return ebiten.Termination
	}

	defer func() {
		d.closeErr = d.guest.Close()
		d.guestSessionClosed = true
	}()

	if d.runScript != nil {
		if err := d.runScript(d); err != nil {
			d.testErr = err
			return ebiten.Termination
		}
	}

	return ebiten.Termination
}

func (d *guiDriver) Draw(screen *ebiten.Image) {}

func (d *guiDriver) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

// captureFrame requests, waits for, and composites a frame from the guest into d.screen,
// then writes the snapshot to a PNG file in d.outDir.
func (d *guiDriver) captureFrame(name string) (image.Image, error) {
	d.guest.AdvanceFrame()
	if !d.guest.WaitFrame() {
		return nil, fmt.Errorf("WaitFrame failed: %w", d.guest.Err())
	}
	if !d.guest.CompositeFrame() {
		return nil, errors.New("CompositeFrame failed")
	}

	b := d.screen.Bounds()
	img := image.NewRGBA(b)
	d.screen.ReadPixels(img.Pix)

	// Validate that the image contains rendered pixels (not completely blank/transparent).
	if isImageBlank(img) {
		return nil, fmt.Errorf("captured frame %q is completely blank", name)
	}

	filePath := filepath.Join(d.outDir, fmt.Sprintf("%s_%s.png", d.scenarioName, name))
	if err := savePNG(filePath, img); err != nil {
		return nil, fmt.Errorf("failed to save snapshot %q: %w", name, err)
	}
	d.snapshots = append(d.snapshots, filePath)
	slog.Info("Captured snapshot", "scenario", d.scenarioName, "name", name, "path", filePath)
	return img, nil
}

// click simulates moving the cursor, pressing the mouse button, advancing ticks, and releasing it.
func (d *guiDriver) click(x, y float64, holdTicks int) {
	d.guest.MoveCursor(x, y)
	d.guest.PressMouseButton(ebiten.MouseButtonLeft)
	d.guest.AdvanceTicks(holdTicks)
	d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
}

func isImageBlank(img *image.RGBA) bool {
	nonTransparent := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 0 {
			nonTransparent++
			if nonTransparent > 100 {
				return false
			}
		}
	}
	return true
}

func imagesDiffer(imgA, imgB image.Image) bool {
	boundsA := imgA.Bounds()
	boundsB := imgB.Bounds()
	if boundsA != boundsB {
		return true
	}
	diffPixels := 0
	for y := boundsA.Min.Y; y < boundsA.Max.Y; y += 4 {
		for x := boundsA.Min.X; x < boundsA.Max.X; x += 4 {
			c1 := color.RGBAModel.Convert(imgA.At(x, y)).(color.RGBA)
			c2 := color.RGBAModel.Convert(imgB.At(x, y)).(color.RGBA)
			if c1 != c2 {
				diffPixels++
				if diffPixels > 50 {
					return true
				}
			}
		}
	}
	return false
}

func savePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func runGuestScenario(sc scenarioConfig, outDir string) error {
	dir, err := os.MkdirTemp("", "s30_gui_test_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	socketPath := filepath.Join(dir, "guest.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer ln.Close()

	endpoint, err := vmhost.EndpointURLFromAddr(ln.Addr())
	if err != nil {
		return err
	}

	guestBin := filepath.Join(dir, "guest_binary")
	buildCmd := exec.Command("go", "build", "-tags", "ebitenginevmguest", "-o", guestBin, sc.pkg)
	buildCmd.Stderr = os.Stderr
	if err = buildCmd.Run(); err != nil {
		return fmt.Errorf("failed to build guest (%s): %w", sc.pkg, err)
	}

	guestCmd := exec.Command(guestBin)
	guestCmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	var guestStderr bytes.Buffer
	guestCmd.Stderr = &guestStderr

	if err = guestCmd.Start(); err != nil {
		return fmt.Errorf("failed to start guest: %w", err)
	}
	var processReaped bool
	defer func() {
		if !processReaped {
			_ = guestCmd.Process.Kill()
			_ = guestCmd.Wait()
		}
	}()

	if dl, ok := ln.(interface{ SetDeadline(time.Time) error }); ok {
		_ = dl.SetDeadline(time.Now().Add(10 * time.Second))
	}
	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accepting guest connection timed out: %w (stderr: %s)", err, guestStderr.String())
	}
	_ = ln.Close()

	driver := &guiDriver{
		screen:       ebiten.NewImage(sc.width, sc.height),
		outDir:       outDir,
		scenarioName: sc.name,
		runScript:    sc.runTest,
	}

	guestSession, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{
		IdleTimeout: 10 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("creating guest session: %w", err)
	}
	driver.guest = guestSession

	ebiten.SetWindowVisible(false)
	runErr := ebiten.RunGame(driver)

	if driver.guestSessionClosed {
		_ = waitForExit(guestCmd)
	} else {
		_ = guestCmd.Process.Kill()
		_ = guestCmd.Wait()
	}
	processReaped = true

	if runErr != nil && !errors.Is(runErr, ebiten.Termination) {
		return fmt.Errorf("host run failed: %w", runErr)
	}
	if driver.testErr != nil {
		return fmt.Errorf("scenario %q assertion failed: %w", sc.name, driver.testErr)
	}
	return nil
}

func waitForExit(cmd *exec.Cmd) error {
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		return errors.New("guest timed out exiting")
	}
}

// Scenarios

func mainGameScenario() scenarioConfig {
	return scenarioConfig{
		name:   "main_game",
		pkg:    ".",
		width:  1024,
		height: 768,
		runTest: func(d *guiDriver) error {
			// 1. Settle on title screen
			d.guest.AdvanceTicks(60)
			frame0, err := d.captureFrame("01_title_screen")
			if err != nil {
				return err
			}

			// 2. Click "New Game" button at centerX=512, Y=400
			d.click(512, 400, 2)
			d.guest.AdvanceTicks(40)

			frame1, err := d.captureFrame("02_difficulty_screen")
			if err != nil {
				return err
			}

			// Verify screen transitioned upon clicking New Game
			if !imagesDiffer(frame0, frame1) {
				return errors.New("screen did not change after clicking 'New Game'")
			}

			// 3. Click difficulty button (e.g. Easy, at centerX=826, slot Y=188)
			d.click(826, 188, 2)
			d.guest.AdvanceTicks(40)

			frame2, err := d.captureFrame("03_color_selection_screen")
			if err != nil {
				return err
			}

			// Verify screen transitioned to color selection
			if !imagesDiffer(frame1, frame2) {
				return errors.New("screen did not change after selecting difficulty")
			}

			slog.Info("Main game GUI navigation test passed successfully!")
			return nil
		},
	}
}

func editDeckScenario() scenarioConfig {
	return scenarioConfig{
		name:   "edit_deck",
		pkg:    "./cmd/edit_deck_test",
		width:  1024,
		height: 768,
		runTest: func(d *guiDriver) error {
			d.guest.AdvanceTicks(60)
			_, err := d.captureFrame("01_deck_view")
			if err != nil {
				return err
			}
			slog.Info("Edit deck GUI test passed successfully!")
			return nil
		},
	}
}

func dungeonScenario() scenarioConfig {
	return scenarioConfig{
		name:   "dungeon",
		pkg:    "./cmd/dungeon_test",
		width:  1024,
		height: 768,
		runTest: func(d *guiDriver) error {
			d.guest.AdvanceTicks(60)
			_, err := d.captureFrame("01_dungeon_view")
			if err != nil {
				return err
			}
			slog.Info("Dungeon GUI test passed successfully!")
			return nil
		},
	}
}

func duelScenario() scenarioConfig {
	return scenarioConfig{
		name:   "duel",
		pkg:    "./cmd/duel_test",
		width:  1024,
		height: 768,
		runTest: func(d *guiDriver) error {
			d.guest.AdvanceTicks(60)
			_, err := d.captureFrame("01_mulligan_view")
			if err != nil {
				return err
			}
			slog.Info("Duel GUI test passed successfully!")
			return nil
		},
	}
}

func main() {
	target := flag.String("test", "all", "test scenario to run: 'main', 'edit_deck', 'dungeon', 'duel', or 'all'")
	outDir := flag.String("out", "dist/gui_test", "directory to save captured GUI snapshots")
	flag.Parse()

	scenarios := map[string]scenarioConfig{
		"main":      mainGameScenario(),
		"edit_deck": editDeckScenario(),
		"dungeon":   dungeonScenario(),
		"duel":      duelScenario(),
	}

	if *target == "all" {
		order := []string{"main", "edit_deck", "dungeon", "duel"}
		fmt.Printf("=== Running Headless GUI Test Suite (%d scenarios) ===\n", len(order))
		exe, err := os.Executable()
		if err != nil {
			exe = "go"
		}

		failed := 0
		for _, name := range order {
			start := time.Now()
			fmt.Printf("--- RUN: %s\n", name)

			var cmd *exec.Cmd
			if strings.HasSuffix(exe, "gui_test") {
				cmd = exec.Command(exe, "-test", name, "-out", *outDir)
			} else {
				cmd = exec.Command("go", "run", "./cmd/gui_test", "-test", name, "-out", *outDir)
			}
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				fmt.Printf("--- FAIL: %s (took %v): %v\n", name, time.Since(start).Round(time.Millisecond), err)
				failed++
			} else {
				fmt.Printf("--- PASS: %s (took %v)\n", name, time.Since(start).Round(time.Millisecond))
			}
		}

		if failed > 0 {
			fmt.Printf("\nFAIL: %d/%d GUI scenarios failed\n", failed, len(order))
			os.Exit(1)
		}
		fmt.Printf("\nPASS: all %d GUI scenarios passed headlessly!\n", len(order))
		return
	}

	sc, ok := scenarios[*target]
	if !ok {
		log.Fatalf("Unknown test scenario: %q (available: main, edit_deck, dungeon, duel, all)", *target)
	}

	if err := runGuestScenario(sc, *outDir); err != nil {
		log.Fatalf("GUI scenario %q failed: %v", *target, err)
	}
}
