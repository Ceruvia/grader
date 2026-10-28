package tasks

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
)

func TestGradingBoundsDiagnostics(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	for _, tc := range []struct {
		name, recipe, program string
		compileFailure        bool
	}{
		{"compile", "head -c 100000 /dev/zero | tr '\\000' x; false", "", true},
		{"invalid UTF8 compile", "head -c 100000 /dev/zero | tr '\\000' '\\377'; false", "", true},
		{"runtime", "gcc -o main main.c", "#include <stdio.h>\nint main(void) { for(int i=0;i<100000;i++) putchar('x'); return 1; }\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"1.in": "", "1.out": "answer\n", "2.in": "", "2.out": "answer\n"},
				"/submission.zip": {"Makefile": "all:\n\t@" + tc.recipe + "\n", "main.c": tc.program},
			})
			result, err := GradeBlackboxWithBuilder("bounded-diagnostics", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in", "2.in"}, []string{"1.out", "2.out"}, 1000, 65536, "C", "Makefile", "", "main")
			if err != nil {
				t.Fatal(err)
			}
			var got evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &got); err != nil {
				t.Fatal(err)
			}
			message := got.ErrorMessage
			if tc.compileFailure {
				if got.IsSuccess || got.Status != "Compile Error" {
					t.Fatalf("expected compile failure, got %+v", got)
				}
			} else {
				if !got.IsSuccess || len(got.TestcaseGradingResult) != 2 || got.TestcaseGradingResult[0].Verdict != models.VerdictRE {
					t.Fatalf("expected RE, got %+v", got)
				}
				message = got.TestcaseGradingResult[0].ErrorMessage
			}
			total := len(got.ErrorMessage)
			for _, result := range got.TestcaseGradingResult {
				total += len(result.ErrorMessage)
			}
			if total > 64<<10 {
				t.Fatalf("task diagnostics total %d bytes, maximum is 64 KiB", total)
			}
			if len(message) == 0 || len(message) > 64<<10 {
				t.Fatalf("diagnostic must be a nonempty prefix of at most 64 KiB, got %d bytes", len(message))
			}
			if !strings.HasPrefix(message, "x") && !strings.HasPrefix(message, "\ufffd") {
				t.Fatalf("lost diagnostic prefix: %.50q", message)
			}
		})
	}
}
