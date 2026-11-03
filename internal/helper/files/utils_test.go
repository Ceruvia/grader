package files_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/files"
)

// A student program can leave a symlink in the box under the name of a file the
// grader copies in next. Root must replace the link, not write through it.
func TestCopyFileReplacesPlantedSymlink(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "case1.in")
	victim := filepath.Join(dir, "victim")
	box := filepath.Join(dir, "box")
	if err := os.WriteFile(src, []byte("teacher input"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(victim, []byte("host secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(box, 0700); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(box, "case1.in")
	if err := os.Symlink(victim, dst); err != nil {
		t.Fatal(err)
	}

	if _, err := files.CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}

	if got, _ := os.ReadFile(victim); string(got) != "host secret" {
		t.Fatalf("CopyFile wrote through the planted symlink: victim now %q", got)
	}
	info, err := os.Lstat(dst)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("dst must be a regular file, got %v, %v", info, err)
	}
	if got, _ := os.ReadFile(dst); string(got) != "teacher input" {
		t.Fatalf("dst = %q, want the teacher input", got)
	}
}
