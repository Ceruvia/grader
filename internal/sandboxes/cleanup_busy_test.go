//go:build linux && integration

package sandboxes_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

// holdBoxFile models a fork in another worker: a child holds a copy of a box
// file the grader has already closed, until release closes the child's stdin.
func holdBoxFile(t *testing.T, sbx *sandboxes.IsolateSandbox) (release func()) {
	t.Helper()
	path := filepath.Join(sbx.GetBoxdir(), "out.txt")
	if err := os.WriteFile(path, []byte("output"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command("sh", "-c", "read _")
	holder.ExtraFiles = []*os.File{f}
	stdin, err := holder.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var once sync.Once
	release = func() {
		once.Do(func() {
			_ = stdin.Close()
			_ = holder.Wait()
		})
	}
	t.Cleanup(release)
	return release
}

func TestCleanupWaitsForForkedChildToReleaseBoxFile(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 842)
	if err != nil {
		t.Fatal(err)
	}
	release := holdBoxFile(t, sbx)
	busy := false
	t.Cleanup(sandboxes.SetOnUnmountBusy(func() {
		busy = true
		release()
	}))
	err = sbx.Cleanup()
	if err != nil {
		t.Fatalf("a holder released after EBUSY must not quarantine the box: %v", err)
	}
	if !busy {
		t.Fatal("the holder did not make unmount report EBUSY")
	}
	if _, err := os.Lstat(sbx.GetBoxdir()); !os.IsNotExist(err) {
		t.Fatalf("box survived cleanup: %v", err)
	}
}

func TestCleanupStillFailsWhileBoxFileIsHeld(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 843)
	if err != nil {
		t.Fatal(err)
	}
	release := holdBoxFile(t, sbx)
	err = sbx.Cleanup()
	release()
	if !errors.Is(err, syscall.EBUSY) {
		t.Fatalf("a held box must fail cleanup so the pool quarantines it, got %v", err)
	}
	if err := sbx.Cleanup(); err != nil {
		t.Fatal(err)
	}
}
