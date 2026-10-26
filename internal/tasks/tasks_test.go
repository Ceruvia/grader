package tasks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestPrepareHelpers(t *testing.T) {
	server := archiveServer(t, map[string]map[string]string{
		"/submission.zip": {"main.c": "int main(void) { return 0; }\n"},
		"/grader.zip":     {"1.in": "input\n", "1.out": "answer\n", "support.h": "#define ANSWER 42\n"},
	})

	t.Run("keeps expected outputs and grader archive outside the box", func(t *testing.T) {
		sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 400)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sbx.Cleanup() })
		taskDir := filepath.Join(t.TempDir(), "task")
		testcases, err := prepareGraderFiles(sbx, taskDir, server.URL+"/grader.zip", []string{"1.in"}, []string{"1.out"})
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(taskDir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0700 {
			t.Fatalf("task directory mode = %o, want 700", info.Mode().Perm())
		}
		expected, err := os.ReadFile(testcases[0].ExpectedOutputPath)
		if err != nil || string(expected) != "answer\n" {
			t.Fatalf("host expected output = %q, error = %v", expected, err)
		}
		for _, name := range []string{"1.out", "grader.zip"} {
			if _, err := os.Lstat(filepath.Join(sbx.GetBoxdir(), name)); !os.IsNotExist(err) {
				t.Errorf("%s entered box, error = %v", name, err)
			}
		}
		for name, want := range map[string]string{"1.in": "input\n", "support.h": "#define ANSWER 42\n"} {
			got, err := os.ReadFile(filepath.Join(sbx.GetBoxdir(), name))
			if err != nil || string(got) != want {
				t.Errorf("%s = %q, error = %v", name, got, err)
			}
		}
	})

	t.Run("downloads submission files into the box", func(t *testing.T) {
		sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 401)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = sbx.Cleanup() })
		sbx.SetTimeLimitInMiliseconds(20000)
		sbx.SetWallTimeLimitInMiliseconds(60000)
		sbx.SetMemoryLimitInKilobytes(1024 * 1024)
		if err := DownloadFileToSandbox(sbx, "submission.zip", server.URL+"/submission.zip"); err != nil {
			t.Fatal(err)
		}
		got, err := sbx.GetFile("main.c")
		if err != nil || string(got) != "int main(void) { return 0; }\n" {
			t.Fatalf("submission source = %q, error = %v", got, err)
		}
		if err := DownloadFileToSandbox(sbx, "invalid.zip", server.URL+"/missing.zip"); err == nil {
			t.Error("expected error for missing archive")
		}
	})
}
