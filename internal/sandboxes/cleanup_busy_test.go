package sandboxes_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

// A fork in another worker briefly holds a copy of a box file the grader has
// already closed. The child here holds it explicitly, for a short and a long time.
func holdBoxFile(t *testing.T, sbx *sandboxes.IsolateSandbox, seconds string) *exec.Cmd {
	t.Helper()
	path := filepath.Join(sbx.GetBoxdir(), "out.txt")
	if err := os.WriteFile(path, []byte("output"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command("sleep", seconds)
	holder.ExtraFiles = []*os.File{f}
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return holder
}

func TestCleanupWaitsForForkedChildToReleaseBoxFile(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 842)
	if err != nil {
		t.Fatal(err)
	}
	holder := holdBoxFile(t, sbx, "0.3")
	defer holder.Wait()
	if err := sbx.Cleanup(); err != nil {
		t.Fatalf("a transient holder must not quarantine the box: %v", err)
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
	holder := holdBoxFile(t, sbx, "30")
	err = sbx.Cleanup()
	_ = holder.Process.Kill()
	_ = holder.Wait()
	if !errors.Is(err, syscall.EBUSY) {
		t.Fatalf("a held box must fail cleanup so the pool quarantines it, got %v", err)
	}
	if err := sbx.Cleanup(); err != nil {
		t.Fatal(err)
	}
}
