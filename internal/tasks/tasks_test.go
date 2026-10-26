package tasks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
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
		testcases, err := prepareGraderFiles(context.Background(), sbx, taskDir, server.URL+"/grader.zip", []string{"1.in"}, []string{"1.out"})
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
		recorder := &commandRecorder{IsolateSandbox: sbx}
		if err := DownloadFileToSandbox(context.Background(), recorder, "submission.zip", server.URL+"/submission.zip"); err != nil {
			t.Fatal(err)
		}
		if len(recorder.commands) != 1 {
			t.Fatalf("expected one unzip run, got %q", recorder.commands)
		}
		// CreateIsolateSandbox allows 50 processes, so isolate takes the memory limit as --cg-mem.
		for _, flag := range []string{" -t10 ", " -w20 ", " --cg-mem=262144 ", " -k262144 "} {
			if !strings.Contains(recorder.commands[0], flag) {
				t.Errorf("unzip command %q lacks limit %q", recorder.commands[0], strings.TrimSpace(flag))
			}
		}
		got, err := sbx.GetFile("main.c")
		if err != nil || string(got) != "int main(void) { return 0; }\n" {
			t.Fatalf("submission source = %q, error = %v", got, err)
		}
		if err := DownloadFileToSandbox(context.Background(), sbx, "invalid.zip", server.URL+"/missing.zip"); err == nil {
			t.Error("expected error for missing archive")
		}
	})
}

// commandRecorder runs the real isolate sandbox and keeps each isolate command line.
type commandRecorder struct {
	*sandboxes.IsolateSandbox
	commands []string
}

func (r *commandRecorder) Execute(run command.CommandBuilder, red sandboxes.RedirectionFiles) sandboxes.SandboxExecutionResult {
	r.commands = append(r.commands, r.BuildCommand(run, red).BuildFullCommand())
	return r.IsolateSandbox.Execute(run, red)
}
