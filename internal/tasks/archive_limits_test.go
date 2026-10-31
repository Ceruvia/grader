package tasks

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestTeacherArchiveLimitsBeforeExtraction(t *testing.T) {
	for _, tooMany := range []bool{false, true} {
		t.Run(fmt.Sprintf("entries=%t", tooMany), func(t *testing.T) {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			entry, err := zw.Create("first.out")
			if err != nil {
				t.Fatal(err)
			}
			if tooMany {
				for i := range 10000 {
					if _, err := zw.Create(fmt.Sprintf("%d.out", i)); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				chunk := make([]byte, 1<<20)
				for range 257 {
					if _, err := entry.Write(chunk); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := zw.Close(); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(buf.Bytes()) }))
			t.Cleanup(server.Close)
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 840)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := sbx.Cleanup(); err != nil {
					t.Error(err)
				}
			})
			taskDir := filepath.Join(t.TempDir(), "task")
			_, err = prepareGraderFiles(context.Background(), sbx, taskDir, server.URL, nil, nil)
			if err == nil {
				t.Error("oversized teacher archive was accepted")
			}
			if _, statErr := os.Lstat(filepath.Join(taskDir, "files", "first.out")); !os.IsNotExist(statErr) {
				t.Errorf("archive must be rejected before extracting its first entry, stat error: %v", statErr)
			}
		})
	}
}
