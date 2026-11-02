//go:build integration

package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestStudentBuildProductsUseEffectiveNames(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 401)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sbx.Cleanup() })

	// Teacher files already in the box must survive even a same-named upload.
	for _, name := range []string{"teacher.o", "teacher.hi", "teacher.gch", "teacher.pyc", "trusted/__pycache__/teacher.pyc"} {
		path := filepath.Join(sbx.GetBoxdir(), name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("teacher-canary"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	entries := map[string]string{
		"Solution.hs":                       "module Solution where\nanswer = 42\n",
		"teacher.o;1":                       "student must not replace the teacher object",
		"teacher.hi":                        "student must not replace the teacher interface",
		"object.o":                          symlinkEntry + "/box/teacher.o",
		"pkg/__pycache__/marker.txt":        "remove the whole student cache directory",
		"trusted/__pycache__/student.pyc;1": "student cache inside a teacher directory",
	}
	for _, ext := range []string{".o", ".hi", ".dyn_o", ".dyn_hi", ".p_o", ".p_hi", ".gch", ".pyc"} {
		for _, suffix := range []string{"", ";1", "\x01"} {
			entries["nested/artifact"+ext+suffix] = "stale student build product"
		}
	}
	server := archiveServer(t, map[string]map[string]string{"/submission.zip": entries})
	if err := DownloadFileToSandbox(context.Background(), sbx, "submission.zip", server.URL+"/submission.zip"); err != nil {
		t.Fatal(err)
	}

	for _, ext := range []string{".o", ".hi", ".dyn_o", ".dyn_hi", ".p_o", ".p_hi", ".gch", ".pyc"} {
		if _, err := os.Lstat(filepath.Join(sbx.GetBoxdir(), "nested/artifact"+ext)); !os.IsNotExist(err) {
			t.Errorf("effective artifact name %s survived: %v", ext, err)
		}
	}
	for _, name := range []string{"object.o", "pkg/__pycache__", "trusted/__pycache__/student.pyc"} {
		if _, err := os.Lstat(filepath.Join(sbx.GetBoxdir(), name)); !os.IsNotExist(err) {
			t.Errorf("student build product %s survived: %v", name, err)
		}
	}
	for _, name := range []string{"teacher.o", "teacher.hi", "teacher.gch", "teacher.pyc", "trusted/__pycache__/teacher.pyc"} {
		got, err := os.ReadFile(filepath.Join(sbx.GetBoxdir(), name))
		if err != nil || string(got) != "teacher-canary" {
			t.Errorf("teacher file %s changed: %q, %v", name, got, err)
		}
	}
	got, err := os.ReadFile(filepath.Join(sbx.GetBoxdir(), "Solution.hs"))
	if err != nil || string(got) != entries["Solution.hs"] {
		t.Fatalf("student source was not preserved: %q, %v", got, err)
	}
}
