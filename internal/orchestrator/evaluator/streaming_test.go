package evaluator_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestEvaluatorBoundsFailureDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		status  sandboxes.SandboxExecutionStatus
		verdict models.Verdict
	}{
		{sandboxes.NONZERO_EXIT_CODE, models.VerdictRE},
		{sandboxes.KILLED_ON_SIGNAL, models.VerdictRE},
		{sandboxes.TIMED_OUT, models.VerdictTLE},
		{sandboxes.INTERNAL_ERROR, models.VerdictXX},
		{sandboxes.PARSING_META_ERROR, models.VerdictXX},
	} {
		t.Run(tc.status.String(), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "actual.out"), []byte(strings.Repeat("\xff", 100000)), 0600); err != nil {
				t.Fatal(err)
			}
			sbx := &sandboxes.IsolateSandbox{BoxDir: dir}
			got := (evaluator.SimpleEvaluator{}).Evaluate(sbx, sandboxes.SandboxExecutionResult{Status: tc.status, Message: strings.Repeat("\xff", 100000)}, "unused", "actual.out", 64<<10)
			if got.Verdict != tc.verdict || !got.HasErrorMessage || len(got.ErrorMessage) == 0 || len(got.ErrorMessage) > 64<<10 {
				t.Fatalf("want %s with bounded nonempty diagnostic, got verdict=%s bytes=%d", tc.verdict.Code, got.Verdict.Code, len(got.ErrorMessage))
			}
		})
	}
}

func TestEvaluatorPreservesExactComparison(t *testing.T) {
	for _, tc := range []struct {
		name, actual, expected string
		verdict                models.Verdict
	}{
		{"empty", "", "", models.VerdictAC},
		{"CRLF across buffer edge", strings.Repeat("a", 4095) + "\r\nb\r\n", strings.Repeat("a", 4095) + "\nb\n", models.VerdictAC},
		{"lone CR stays significant", "a\rb", "a\nb", models.VerdictWA},
		{"trailing newline stays significant", "answer\n", "answer", models.VerdictWA},
		{"large equal", strings.Repeat("abc\n", 20000), strings.Repeat("abc\n", 20000), models.VerdictAC},
		{"long prefix differs at EOF", strings.Repeat("a", 70000), strings.Repeat("a", 70000) + "b", models.VerdictWA},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, body := range map[string]string{"actual.out": tc.actual, "expected.out": tc.expected} {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			sbx := &sandboxes.IsolateSandbox{BoxDir: dir}
			got := (evaluator.SimpleEvaluator{}).Evaluate(sbx, sandboxes.SandboxExecutionResult{Status: sandboxes.ZERO_EXIT_CODE}, filepath.Join(dir, "expected.out"), "actual.out", 64<<10)
			if got.Verdict != tc.verdict {
				t.Fatalf("want %s, got %+v", tc.verdict.Code, got)
			}
		})
	}
}
