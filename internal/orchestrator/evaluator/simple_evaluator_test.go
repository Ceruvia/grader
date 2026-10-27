package evaluator_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

// Isolate 2.1 deletes symlinks and FIFOs from the box after each run. These tests plant
// them from the host so the evaluator stays safe without relying on that cleanup.
func TestEvaluatorRejectsUnsafeOutput(t *testing.T) {
	for _, test := range []struct {
		name  string
		plant func(t *testing.T, actualPath, hostFile string)
	}{
		{"symlink to host file", func(t *testing.T, actualPath, hostFile string) {
			if err := os.Symlink(hostFile, actualPath); err != nil {
				t.Fatal(err)
			}
		}},
		{"fifo", func(t *testing.T, actualPath, _ string) {
			if err := syscall.Mkfifo(actualPath, 0666); err != nil {
				t.Fatal(err)
			}
		}},
		{"larger than the box file size limit", func(t *testing.T, actualPath, _ string) {
			if err := os.WriteFile(actualPath, bytes.Repeat([]byte("x"), 2048), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			expected := filepath.Join(dir, "expected.out")
			if err := os.WriteFile(expected, []byte("known-answer\n"), 0600); err != nil {
				t.Fatal(err)
			}
			hostFile := filepath.Join(dir, "host-file")
			if err := os.WriteFile(hostFile, []byte("root:host-file-canary\n"), 0600); err != nil {
				t.Fatal(err)
			}
			boxDir := filepath.Join(dir, "box")
			if err := os.Mkdir(boxDir, 0700); err != nil {
				t.Fatal(err)
			}
			test.plant(t, filepath.Join(boxDir, "1.out.actual"), hostFile)
			sbx := &sandboxes.IsolateSandbox{BoxDir: boxDir, FileSizeLimit: 1}
			sbx.AddFileWithoutMove("1.out.actual")

			done := make(chan evaluator.EngineRunResult, 1)
			go func() {
				done <- (evaluator.SimpleEvaluator{}).Evaluate(sbx, sandboxes.SandboxExecutionResult{Status: sandboxes.ZERO_EXIT_CODE}, expected, "1.out.actual", 64<<10)
			}()
			var result evaluator.EngineRunResult
			select {
			case result = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("evaluator blocked reading the actual output")
			}
			if result.Verdict != models.VerdictRE || strings.Contains(result.ErrorMessage, "root:") {
				t.Fatalf("expected safe Runtime Error, got %+v", result)
			}
		})
	}
}
