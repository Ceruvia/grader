package tasks

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
)

// symlinkEntry marks an archive entry as a symlink to the rest of the content, as `zip --symlinks` stores it.
const symlinkEntry = "symlink:"

// prebuiltEntry marks an executable archive entry whose mtime is a day in the future.
const prebuiltEntry = "prebuilt:"

func archiveServer(t *testing.T, archives map[string]map[string]string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for path, entries := range archives {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for name, content := range entries {
			header := &zip.FileHeader{Name: name, Method: zip.Deflate}
			if target, ok := strings.CutPrefix(content, symlinkEntry); ok {
				header.SetMode(os.ModeSymlink | 0777)
				content = target
			} else if body, ok := strings.CutPrefix(content, prebuiltEntry); ok {
				header.SetMode(0755)
				header.Modified = time.Now().Add(24 * time.Hour)
				content = body
			}
			entry, err := zw.CreateHeader(header)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := entry.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		data := bytes.Clone(buf.Bytes())
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(data)
		})
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func TestGradingKeepsExpectedOutputsOnHost(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)

	for _, builder := range []bool{false, true} {
		mode := "source"
		if builder {
			mode = "makefile"
		}
		for _, test := range []struct {
			name    string
			program string
			verdict models.Verdict
		}{
			{
				name:    "honest",
				program: "#include <stdio.h>\nint main(void) { puts(\"known-answer-canary\"); return 0; }\n",
				verdict: models.VerdictAC,
			},
			{
				name:    "reads_expected_output",
				program: "#include <stdio.h>\nint main(void) { FILE *f = fopen(\"1.out\", \"r\"); if (f) { int c; while ((c = fgetc(f)) != EOF) putchar(c); fclose(f); } return 0; }\n",
				verdict: models.VerdictWA,
			},
			{
				name:    "symlinks_actual_output",
				program: "#include <unistd.h>\nint main(void) { unlink(\"1.out.actual\"); if (symlink(\"/etc/passwd\", \"1.out.actual\")) return 2; return 1; }\n",
				verdict: models.VerdictRE,
			},
		} {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				server := archiveServer(t, map[string]map[string]string{
					"/grader.zip": {"1.in": "", "1.out": "known-answer-canary\n"},
					"/submission.zip": {
						"main.c":   test.program,
						"Makefile": "all:\n\tgcc main.c -o main\n",
					},
				})
				var result string
				var err error
				if builder {
					result, err = GradeBlackboxWithBuilder("security-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "", "main")
				} else {
					result, err = GradeBlackbox("security-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "main.c")
				}
				if err != nil {
					t.Fatal(err)
				}
				var graded evaluator.GradingResult
				if err := json.Unmarshal([]byte(result), &graded); err != nil {
					t.Fatal(err)
				}
				if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 {
					t.Fatalf("expected one executed testcase, got %s", result)
				}
				if got := graded.TestcaseGradingResult[0].Verdict; got != test.verdict {
					t.Errorf("expected %s, got %s: %s", test.verdict, got, result)
				}
				if strings.Contains(result, "root:") {
					t.Error("grading result exposes host /etc/passwd")
				}
			})
		}
	}
}

func TestGradingRestoresInputsBetweenCases(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {"1.in": "1\n", "1.out": "1\n", "2.in": "2\n", "2.out": "2\n"},
		"/submission.zip": {
			"main.c": "#include <stdio.h>\nint main(void) { int n; if (scanf(\"%d\", &n) != 1) return 1; printf(\"%d\\n\", n); if (n == 1) { FILE *f = fopen(\"2.in\", \"w\"); if (!f) return 2; fputs(\"999\\n\", f); fclose(f); } return 0; }\n",
		},
	})
	result, err := GradeBlackbox("fresh-input-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in", "2.in"}, []string{"1.out", "2.out"}, 1000, 65536, "C", "main.c")
	if err != nil {
		t.Fatal(err)
	}
	var graded evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &graded); err != nil {
		t.Fatal(err)
	}
	if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 2 {
		t.Fatalf("expected two executed testcases, got %s", result)
	}
	for _, tc := range graded.TestcaseGradingResult {
		if tc.Verdict != models.VerdictAC {
			t.Errorf("fresh input should produce AC, got %s", result)
		}
	}
}

func TestCompileOutputSymlinkIsNotFollowed(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	tester.PlantGraderEnv(t, "BACKEND_API_TOKEN=env-file-canary\n")
	// The pool hands out box 1, so the host keeps the expected outputs in /var/lib/grader/tasks/1/files.
	for name, target := range map[string]string{
		"grader_env":      tester.GraderEnvPath,
		"expected_output": "/var/lib/grader/tasks/1/files/1.out",
	} {
		t.Run(name, func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				"/grader.zip": {"1.in": "", "1.out": "known-answer-canary\n"},
				"/submission.zip": {
					"Makefile": "all:\n\tln -sf " + target + " _compile.out; exit 1\n",
				},
			})
			result, err := GradeBlackboxWithBuilder("compile-symlink-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "", "main")
			if err != nil {
				t.Fatal(err)
			}
			var graded evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &graded); err != nil {
				t.Fatal(err)
			}
			if graded.IsSuccess {
				t.Fatalf("expected a failed compile, got %s", result)
			}
			for _, canary := range []string{"env-file-canary", "known-answer-canary"} {
				if strings.Contains(result, canary) {
					t.Fatalf("compile result exposes host file content %q: %s", canary, result)
				}
			}
		})
	}
}

func TestActualOutputFifoDoesNotBlockGrading(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {"1.in": "", "1.out": "known-answer-canary\n"},
		"/submission.zip": {
			"main.c": "#include <sys/stat.h>\n#include <unistd.h>\nint main(void) { unlink(\"1.out.actual\"); return mkfifo(\"1.out.actual\", 0666) ? 2 : 0; }\n",
		},
	})
	type outcome struct {
		result string
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := GradeBlackbox("fifo-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "main.c")
		done <- outcome{result, err}
	}()
	var got outcome
	select {
	case got = <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("grading blocked on a FIFO planted as the actual output")
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	var graded evaluator.GradingResult
	if err := json.Unmarshal([]byte(got.result), &graded); err != nil {
		t.Fatal(err)
	}
	if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 {
		t.Fatalf("expected one executed testcase, got %s", got.result)
	}
	if v := graded.TestcaseGradingResult[0].Verdict; v != models.VerdictRE && v != models.VerdictWA {
		t.Fatalf("expected RE or WA, got %s: %s", v, got.result)
	}
}

func TestSubmissionSymlinksDoNotReachHostFiles(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)

	const canaryContent = "host-canary\n"
	links := []string{"_compile.meta", "_compile.out", "1.out.meta", "1.out.actual"}
	submission := map[string]string{
		"main.c": "#include <stdio.h>\nint main(void) { int x; if (scanf(\"%d\", &x) == 1) printf(\"%d\\n\", x * 2); return 0; }\n",
	}
	canaries := map[string]string{}
	for _, link := range links {
		canary := filepath.Join(os.TempDir(), "canary-"+strconv.FormatInt(time.Now().UnixNano(), 36)+"-"+link)
		if err := os.WriteFile(canary, []byte(canaryContent), 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(canary) })
		canaries[link] = canary
		submission[link] = symlinkEntry + canary
	}
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip":     {"1.in": "21\n", "1.out": "42\n"},
		"/submission.zip": submission,
	})

	result, err := GradeBlackbox("symlink-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "main.c")
	if err != nil {
		t.Fatal(err)
	}
	for link, canary := range canaries {
		got, err := os.ReadFile(canary)
		if err != nil || string(got) != canaryContent {
			t.Errorf("host file behind %s changed: %q, %v", link, got, err)
		}
	}
	var graded evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &graded); err != nil {
		t.Fatal(err)
	}
	if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 || graded.TestcaseGradingResult[0].Verdict != models.VerdictAC {
		t.Fatalf("expected the honest program to get AC, got %s", result)
	}
}
