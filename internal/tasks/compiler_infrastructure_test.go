package tasks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/pool"
)

func TestSubmitCompilerInfrastructureFailure(t *testing.T) {
	// Fault injection at the isolate executable boundary. Unzip/init/cleanup still
	// use real isolate, but compiler invocations simulate isolate's internal exit 2.
	wrapper := filepath.Join(t.TempDir(), "isolate-with-compiler-failure")
	script := "#!/bin/sh\nfor arg do\n case \"$arg\" in /usr/bin/gcc|/usr/bin/make) exit 2;; esac\ndone\nexec /usr/local/bin/isolate \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := pool.NewSandboxPool(wrapper, 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip":     {"1.in": "", "1.out": "answer\n"},
		"/submission.zip": {"main.c": "int main(void) { return 0; }\n", "Makefile": "all:\n\tgcc -o main main.c\n"},
	})
	for _, builder := range []bool{false, true} {
		var result string
		var err error
		if builder {
			result, err = GradeBlackboxWithBuilder("compiler-infrastructure", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "", "main")
		} else {
			result, err = GradeBlackbox("compiler-infrastructure", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "main.c")
		}
		if err == nil || result != "" || !strings.Contains(strings.ToLower(err.Error()), "internal") {
			t.Fatalf("builder=%t: infrastructure failure must fail the Machinery task (Backend maps FAILURE to XX), got result=%s error=%v", builder, result, err)
		}
	}
}
