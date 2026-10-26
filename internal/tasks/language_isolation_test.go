package tasks

import (
	"encoding/json"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
)

func TestSandboxLanguageToolchains(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	for _, test := range []struct {
		language string
		filename string
		source   string
	}{
		{"C", "main.c", "#include <stdio.h>\nint main(void) { puts(\"42\"); return 0; }\n"},
		{"C++17", "main.cpp", "#include <iostream>\nint main() { std::cout << 42 << '\\n'; }\n"},
		{"Python 3", "main.py", "print(42)\n"},
		{"Haskell", "main.hs", "main = print (42 :: Int)\n"},
	} {
		t.Run(test.language, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"1.in": "", "1.out": "42\n"},
				"/submission.zip": {test.filename: test.source},
			})
			result, err := GradeBlackbox("language-isolation-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, test.language, test.filename)
			if err != nil {
				t.Fatal(err)
			}
			var graded evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &graded); err != nil {
				t.Fatal(err)
			}
			if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 || graded.TestcaseGradingResult[0].Verdict != models.VerdictAC {
				t.Fatalf("expected successful compile and AC, got %s", result)
			}
		})
	}
}
