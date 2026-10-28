package tasks

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestCompileCheck(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)

	for _, tc := range []struct {
		name, status, contains string
		builder                bool
		submission             map[string]string
	}{
		{
			name: "valid C with teacher header", status: "compiled",
			submission: map[string]string{"main.c": "#include \"support.h\"\nint main(void) { return ANSWER; }\n"},
		},
		{
			name: "gcc syntax error", status: "compile_error", contains: "error:",
			submission: map[string]string{"main.c": "int main(void) { return ; syntax error }\n"},
		},
		{
			// There are no .in/.out files. If this were run, it would emit wrong
			// output and fail. Test must only compile it, regardless of output.
			name: "wrong output is not executed", status: "compiled",
			submission: map[string]string{"main.c": "#include <stdio.h>\nint main(void) { puts(\"wrong answer\"); for (;;) {} }\n"},
		},
		{
			name: "Makefile failure", builder: true, status: "compile_error", contains: "compile-failed-canary",
			submission: map[string]string{"Makefile": "all:\n\t@echo compile-failed-canary >&2\n\t@exit 1\n"},
		},
		{
			name: "Makefile success", builder: true, status: "compiled",
			submission: map[string]string{"main.c": "int main(void) { return 0; }\n", "Makefile": "all:\n\tgcc -o main main.c\n"},
		},
		{
			name: "diagnostics capped", builder: true, status: "compile_error", contains: "diagnostic-canary",
			submission: map[string]string{"Makefile": "all:\n\t@echo diagnostic-canary; head -c 70000 /dev/zero | tr '\\000' x; exit 1\n"},
		},
		{
			name: "invalid UTF8 diagnostics capped", builder: true, status: "compile_error", contains: "\ufffd",
			submission: map[string]string{"Makefile": "all:\n\t@head -c 70000 /dev/zero | tr '\\000' '\\377'; exit 1\n"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"support.h": "#define ANSWER 42\n"},
				"/submission.zip": tc.submission,
			})
			var result string
			var err error
			if tc.builder {
				result, err = CompileCheckWithBuilder("compile-only", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "Makefile", "", 0)
			} else {
				result, err = CompileCheck("compile-only", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "main.c", 0)
			}
			if err != nil {
				t.Fatal(err)
			}
			var got compileCheckResult
			if err := json.Unmarshal([]byte(result), &got); err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.status || !strings.Contains(got.Message, tc.contains) {
				t.Fatalf("expected %s containing %q, got %s", tc.status, tc.contains, result)
			}
			if len(got.Message) > 64<<10 {
				t.Fatalf("message has %d bytes, must be at most 64 KiB", len(got.Message))
			}
			if tc.name == "diagnostics capped" && len(got.Message) != 64<<10 {
				t.Fatalf("expected diagnostic prefix of 64 KiB, got %d bytes", len(got.Message))
			}
			if pool.Pool.IdleCount() != 1 {
				t.Error("compile check did not release its sandbox")
			}
			if _, err := os.Lstat(sandboxes.TaskDir(1)); !os.IsNotExist(err) {
				t.Fatalf("task directory was not cleaned up: %v", err)
			}
		})
	}
}
