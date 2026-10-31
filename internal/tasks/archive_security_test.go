package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestTeacherArchiveRejectsUnsafePaths(t *testing.T) {
	for _, tc := range []struct{ name, path, body string }{
		{"parent", "../escape.out", "escaped"},
		{"absolute", filepath.Join(t.TempDir(), "escape.out"), "escaped"},
		{"symlink", "escape", symlinkEntry + ".."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip": {"first.out": "must not be extracted", tc.path: tc.body},
			})
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 841)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := sbx.Cleanup(); err != nil {
					t.Error(err)
				}
			})
			taskDir := filepath.Join(t.TempDir(), "task")
			if _, err := prepareGraderFiles(context.Background(), sbx, taskDir, server.URL+"/grader.zip", nil, nil); err == nil {
				t.Error("unsafe teacher archive was accepted")
			}
			if _, err := os.Lstat(filepath.Join(taskDir, "files", "first.out")); !os.IsNotExist(err) {
				t.Fatalf("unsafe archive must be rejected before extracting any file, got stat error %v", err)
			}
		})
	}
}
