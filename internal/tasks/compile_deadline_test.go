package tasks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/pool"
)

func TestCompileCheckDeadline(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	previousClock, previousPool := compileCheckNow, pool.Pool
	compileCheckNow = func() time.Time { return now }
	t.Cleanup(func() { compileCheckNow, pool.Pool = previousClock, previousPool })

	for _, tc := range []struct {
		name     string
		deadline int64
		expired  bool
	}{
		{"expired", now.UnixMilli() - 1, true},
		{"future", now.Add(90 * time.Second).UnixMilli(), false},
		{"at deadline", now.UnixMilli(), false},
		{"zero", 0, false},
		{"negative", -1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expired {
				// Any sandbox acquisition panics with a nil pool, proving early rejection.
				pool.Pool = nil
			} else {
				if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(pool.Pool.Close)
			}
			archives := archiveServer(t, map[string]map[string]string{
				"/grader.zip":     {"support.h": "#define ANSWER 42\n"},
				"/submission.zip": {"main.c": "#include \"support.h\"\nint main(void) { return ANSWER; }\n", "Makefile": "all:\n\tgcc -o main main.c\n"},
			})
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				archives.Config.Handler.ServeHTTP(w, r)
			}))
			t.Cleanup(server.Close)
			for _, builder := range []bool{false, true} {
				var result string
				var err error
				if builder {
					result, err = CompileCheckWithBuilder("deadline-test", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "Makefile", "", tc.deadline)
				} else {
					result, err = CompileCheck("deadline-test", server.URL+"/grader.zip", server.URL+"/submission.zip", "C", "main.c", tc.deadline)
				}
				if err != nil {
					t.Fatal(err)
				}
				var got compileCheckResult
				if err := json.Unmarshal([]byte(result), &got); err != nil {
					t.Fatal(err)
				}
				want := compileCheckResult{Status: "compiled"}
				if tc.expired {
					want = compileCheckResult{Status: "internal_error", Message: "Test could not run. Please try again."}
				}
				if got != want {
					t.Fatalf("builder=%t: got %s, want %+v", builder, result, want)
				}
			}
			if tc.expired && requests.Load() != 0 {
				t.Fatalf("expired compile check made %d download requests", requests.Load())
			}
		})
	}
}
