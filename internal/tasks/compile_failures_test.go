package tasks

import (
	"encoding/json"
	"testing"

	"github.com/Ceruvia/grader/internal/pool"
)

func TestCompileCheckStudentFailures(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	for _, tc := range []struct{ name, recipe string }{
		{"wall timeout", "sleep 100"},
		{"removed diagnostic output", "rm -f _compile.out; false"},
		{"nonregular diagnostic output", "rm -f _compile.out; mkfifo _compile.out; false"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"support.h": "// teacher helper\n"},
				"/submission.zip": {"Makefile": "all:\n\t@" + tc.recipe + "\n"},
			})
			result, err := CompileCheckWithBuilder("student-failure-test", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "Makefile", "")
			if err != nil {
				t.Fatal(err)
			}
			var got compileCheckResult
			if err := json.Unmarshal([]byte(result), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != "compile_error" || got.Message == "" {
				t.Fatalf("student compilation failure must be compile_error with diagnostics, got %s", result)
			}
		})
	}
}
