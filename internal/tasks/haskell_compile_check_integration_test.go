//go:build integration

package tasks

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/pool"
)

func TestHaskellCompileCheck(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)

	// Backend e2e/demo/haskell-factorial: the student supplies only Solution.hs.
	const answer = "module Solution (solve) where\n\nsolve :: Integer -> Integer\nsolve n = product [1 .. n]\n"
	const broken = "module Solution (solve) where\n\nsolve :: Integer -> Integer\nsolve n = product [1..n"
	teacher := map[string]string{
		"makefile": "compile:\n\tghc -O0 -o prog Main.hs Solution.hs\n",
		"Main.hs":  "module Main where\n\nimport Solution (solve)\n\nmain :: IO ()\nmain = do\n  n <- readLn\n  print (solve n)\n",
	}

	// Real build products from a prior correct answer, uploaded with edited source.
	buildDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(buildDir, "Solution.hs"), []byte(answer), 0600); err != nil {
		t.Fatal(err)
	}
	ghc := exec.Command("ghc", "-O0", "-c", "Solution.hs")
	ghc.Dir = buildDir
	if out, err := ghc.CombinedOutput(); err != nil {
		t.Fatalf("building stale products: %v: %s", err, out)
	}
	products := make(map[string]string)
	for _, name := range []string{"Solution.o", "Solution.hi"} {
		data, err := os.ReadFile(filepath.Join(buildDir, name))
		if err != nil {
			t.Fatal(err)
		}
		products[name] = string(data)
	}

	for _, path := range []struct {
		name    string
		builder bool
	}{{"without_builder", false}, {"with_builder", true}} {
		t.Run(path.name, func(t *testing.T) {
			for _, tc := range []struct {
				name, source, status string
				stale                bool
			}{
				{"correct_answer", answer, "compiled", false},
				{"syntax_error", broken, "compile_error", false},
				{"syntax_error_with_stale_products", broken, "compile_error", true},
			} {
				t.Run(tc.name, func(t *testing.T) {
					submission := map[string]string{"Solution.hs": tc.source}
					if tc.stale {
						for name, content := range products {
							submission[name] = content
						}
					}
					// archiveServer zeroes mtimes just like Backend's zip.Writer.Create:
					// GHC 8.8 otherwise trusts the same-mtime .o/.hi and skips the .hs.
					server := archiveServer(t, map[string]map[string]string{
						"/grader.zip": teacher, "/submission.zip": submission,
					})
					var result string
					var err error
					if path.builder {
						result, err = CompileCheckWithBuilder("haskell-compile", server.URL+"/grader.zip", server.URL+"/submission.zip", "Haskell", "Makefile", "compile", 0)
					} else {
						result, err = CompileCheck("haskell-compile", server.URL+"/grader.zip", server.URL+"/submission.zip", "Haskell", "Main.hs", 0)
					}
					if err != nil {
						t.Fatal(err)
					}
					var got compileCheckResult
					if err := json.Unmarshal([]byte(result), &got); err != nil {
						t.Fatal(err)
					}
					if got.Status != tc.status {
						t.Fatalf("want %s, got %s", tc.status, result)
					}
					if tc.status == "compile_error" && (!strings.Contains(got.Message, "Solution.hs:") || !strings.Contains(got.Message, "parse error")) {
						t.Fatalf("want Solution.hs compiler parse error, got %s", result)
					}
				})
			}
		})
	}
}
