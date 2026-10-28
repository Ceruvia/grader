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

func TestCppStandardsSelectFlags(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	// std::pair{1, 2} needs class template argument deduction and the binding needs
	// structured bindings, both C++17. Under -std=c++11 the build fails.
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {"1.in": "", "1.out": "3\n"},
		"/submission.zip": {
			"main.cpp": "#include <cstdio>\n#include <utility>\nint main() { auto [a, b] = std::pair{1, 2}; std::printf(\"%d\\n\", a + b); }\n",
		},
	})
	for _, test := range []struct {
		language string
		compiles bool
	}{
		{"C++11", false},
		{"C++17", true},
		{"C++20", true},
	} {
		t.Run(test.language, func(t *testing.T) {
			result, err := GradeBlackbox("cpp-standard-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, test.language, "main.cpp")
			if err != nil {
				t.Fatal(err)
			}
			var graded evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &graded); err != nil {
				t.Fatal(err)
			}
			if !test.compiles {
				if graded.IsSuccess || graded.Status != "Compile Error" {
					t.Fatalf("expected Compile Error, got %s", result)
				}
				return
			}
			if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 || graded.TestcaseGradingResult[0].Verdict != models.VerdictAC {
				t.Fatalf("expected AC, got %s", result)
			}
		})
	}

	// __cplusplus pins the exact standard, so a name mapped to any other flag gets WA.
	// Values observed with g++ 11.4.0 in the grader image.
	printStandard := map[string]string{
		"main.cpp": "#include <cstdio>\nint main() { std::printf(\"%ld\\n\", (long)__cplusplus); }\n",
	}
	for _, test := range []struct {
		language string
		path     string
		want     string
	}{
		{"C++11", "/cpp11", "201103\n"},
		{"C++17", "/cpp17", "201703\n"},
		{"C++20", "/cpp20", "202002\n"},
	} {
		t.Run(test.language+"_cplusplus", func(t *testing.T) {
			server := archiveServer(t, map[string]map[string]string{
				test.path + "/grader.zip":     {"1.in": "", "1.out": test.want},
				test.path + "/submission.zip": printStandard,
			})
			result, err := GradeBlackbox("cpp-standard-value-test", server.URL+test.path+"/grader.zip", server.URL+test.path+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, test.language, "main.cpp")
			if err != nil {
				t.Fatal(err)
			}
			var graded evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &graded); err != nil {
				t.Fatal(err)
			}
			if !graded.IsSuccess || len(graded.TestcaseGradingResult) != 1 || graded.TestcaseGradingResult[0].Verdict != models.VerdictAC {
				t.Fatalf("expected AC for __cplusplus %s, got %s", test.want, result)
			}
		})
	}
}
