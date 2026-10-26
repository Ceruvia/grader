package tasks

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
)

func TestCompileCheckDoesNotExposeHiddenInputs(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	const canary = "hidden-input-secret-canary"
	for _, builder := range []bool{false, true} {
		name := "include input"
		submission := map[string]string{"main.c": "#include \"1.in\"\nint main(void) { return 0; }\n"}
		if builder {
			name = "Makefile reads all inputs"
			submission = map[string]string{"Makefile": "all:\n\t@cat *.in; false\n"}
		}
		t.Run(name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"1.in": "#error " + canary + "\n", "1.out": "answer\n"},
				"/submission.zip": submission,
			})
			var result string
			var err error
			if builder {
				result, err = CompileCheckWithBuilder("hidden-input-test", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "Makefile", "")
			} else {
				result, err = CompileCheck("hidden-input-test", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "main.c")
			}
			if err != nil {
				t.Fatal(err)
			}
			var got compileCheckResult
			if err := json.Unmarshal([]byte(result), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != "compile_error" || strings.Contains(result, canary) {
				t.Fatalf("expected compile_error without hidden input, got %s", result)
			}
		})
	}
}

func TestGradingRemovesPreviousInput(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip":     {"1.in": "1\n", "1.out": "1\n", "2.in": "2\n", "2.out": "2\n"},
		"/submission.zip": {"main.c": "#include <stdio.h>\nint main(void) { int n; if (scanf(\"%d\", &n) != 1) return 1; if (n == 2) { FILE *f = fopen(\"1.in\", \"r\"); if (f) { puts(\"previous input readable\"); fclose(f); return 0; } } printf(\"%d\\n\", n); return 0; }\n"},
	})
	result, err := GradeBlackbox("previous-input-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in", "2.in"}, []string{"1.out", "2.out"}, 1000, 65536, "C", "main.c")
	if err != nil {
		t.Fatal(err)
	}
	var got evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsSuccess || len(got.TestcaseGradingResult) != 2 {
		t.Fatalf("expected two testcases, got %s", result)
	}
	for _, tc := range got.TestcaseGradingResult {
		if tc.Verdict != models.VerdictAC {
			t.Fatalf("expected AC with previous input removed, got %s", result)
		}
	}
}

func TestCompileCheckDownloadErrorIsGeneric(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	// A real HTTP transport failure produces *url.Error including the request URL.
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateNew {
			_ = conn.Close()
		}
	}
	server.Start()
	defer server.Close()
	const canary = "presigned-query-secret-canary"
	result, err := CompileCheck("download-error-test", server.URL+"/grader.zip?signature="+canary, server.URL+"/submission.zip", "C", "main.c")
	if err != nil {
		t.Fatal(err)
	}
	var got compileCheckResult
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatal(err)
	}
	if got.Status != "internal_error" || got.Message != "Test could not run. Please try again." || strings.Contains(result, canary) {
		t.Fatalf("expected generic internal_error without request URL, got %s", result)
	}
}
