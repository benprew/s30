package updates

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestApplyWaitsForExitAndRestarts(t *testing.T) {
	dir := t.TempDir()
	stage, err := os.MkdirTemp(dir, ".s30-update-")
	if err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	target := filepath.Join(dir, "game"+suffix)
	if err = os.WriteFile(target, []byte("old game"), 0755); err != nil {
		t.Fatal(err)
	}
	binary, err := os.ReadFile(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(stage, "s30"+suffix), binary, 0755); err != nil {
		t.Fatal(err)
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	release, key := signedRelease(t, platform, binary, []byte("helper"))
	parent := exec.Command(os.Args[0], "-test.run=^TestUpdateParentProcess$")
	parent.Env = append(os.Environ(), "S30_TEST_UPDATE_PARENT=1")
	if err = parent.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { parent.Process.Kill() })
	oldHash, err := fileChecksum(target)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "restarted")
	plan := installPlan{Release: *release, Platform: platform, Target: target, ParentPID: parent.Process.Pid, OldSHA256: oldHash,
		Args: []string{"-test.run=^TestUpdatedGameProcess$"}, Workdir: dir}
	data, _ := json.Marshal(plan)
	planPath := filepath.Join(stage, "plan.json")
	if err = os.WriteFile(planPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	helper := exec.Command(os.Args[0], "-test.run=^TestApplyHelperProcess$")
	helper.Env = append(os.Environ(), "S30_TEST_APPLY_PLAN="+planPath, "S30_TEST_UPDATE_KEY="+key, "S30_TEST_RESTART_MARKER="+marker)
	if err = helper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { helper.Process.Kill() })
	waitForFile(t, filepath.Join(stage, "ready"))
	installed, _ := os.ReadFile(target)
	if string(installed) != "old game" {
		t.Fatal("game changed before the parent exited")
	}
	parent.Process.Kill()
	if err = parent.Wait(); err == nil {
		t.Fatal("parent was not stopped")
	}
	if err = helper.Wait(); err != nil {
		t.Fatal(err)
	}
	waitForFile(t, marker)
	workdir, _ := os.ReadFile(marker)
	if string(workdir) != dir {
		t.Fatalf("restarted in %q, want %q", workdir, dir)
	}
	digest, err := fileChecksum(target)
	if err != nil || digest != testArtifact("s30", binary).SHA256 {
		t.Fatalf("installed checksum = %s, %v", digest, err)
	}
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("file was not created: %s", path)
}

func TestUpdateParentProcess(t *testing.T) {
	if os.Getenv("S30_TEST_UPDATE_PARENT") != "1" {
		return
	}
	time.Sleep(30 * time.Second)
}

func TestApplyHelperProcess(t *testing.T) {
	path := os.Getenv("S30_TEST_APPLY_PLAN")
	if path == "" {
		return
	}
	PublicKey = os.Getenv("S30_TEST_UPDATE_KEY")
	if err := Apply(path); err != nil {
		t.Fatal(err)
	}
}

func TestUpdatedGameProcess(t *testing.T) {
	marker := os.Getenv("S30_TEST_RESTART_MARKER")
	if marker == "" {
		return
	}
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marker, []byte(dir), 0600); err != nil {
		t.Fatal(err)
	}
}
