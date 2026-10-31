package tasks

import (
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
	"github.com/Ceruvia/grader/internal/pool"
)

func TestGradingBoundsRetainedDiagnostics(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip":     {"1.in": "", "1.out": "", "2.in": "", "2.out": "", "3.in": "", "3.out": ""},
		"/submission.zip": {"main.c": "#include <stdio.h>\nint main(void) { for(int i=0;i<100000;i++) putchar('x'); return 1; }\n"},
	})
	task, err := prepareTask("retained-budget", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in", "2.in", "3.in"}, []string{"1.out", "2.out", "3.out"})
	if err != nil {
		t.Fatal(err)
	}
	defer task.cleanup()
	result := orchestrator.GradeBlackboxSubmission(task.sbx, &models.SubmissionWithFiles{
		Core:               models.Core{Id: "retained-budget", Language: "C", Limits: models.GradingLimit{TimeInMiliseconds: 1000, MemoryInKilobytes: 65536}, Testcases: task.testcases},
		MainSourceFilename: "main.c",
	})
	if !result.IsSuccess || len(result.TestcaseGradingResult) != 3 {
		t.Fatalf("expected three executed cases, got %+v", result)
	}
	retained := 0
	for i, tc := range result.TestcaseGradingResult {
		if tc.Verdict != models.VerdictRE {
			t.Fatalf("case %d lost its verdict after budget exhaustion: %+v", i, tc)
		}
		retained += len(tc.ErrorMessage)
	}
	// This inspects the actual accumulated grading result BEFORE task encoding.
	if retained != 64<<10 {
		t.Fatalf("retained diagnostics must total exactly the 64 KiB prefix budget, got %d bytes", retained)
	}
}
