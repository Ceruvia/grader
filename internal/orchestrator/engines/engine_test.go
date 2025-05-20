package engines_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/engines"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

var Limit = models.GradingLimit{TimeInMiliseconds: 1000, MemoryInKilobytes: 102400}

func TestBlackboxGradingEngine(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		sbx := &sandboxes.IsolateSandbox{
			BoxDir: "../../../tests/not_commited",
		}

		t.Run("Returns BlackBoxGradingEngine", func(t *testing.T) {
			engine, err := engines.CreateBlackboxGradingEngine(
				sbx,
				languages.CGradingLanguage,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello.c",
			)

			tester.AssertNotError(t, err)

			want := engines.BlackboxGradingEngine{
				Sandbox:                    sbx,
				LanguageOrBuilder:          languages.CGradingLanguage,
				Evaluator:                  evaluator.SimpleEvaluator{},
				ExecutableScriptOrFilename: "hello",
			}

			tester.AssertDeep(t, *engine, want)
		})

		t.Run("Returns error when language doesn't exist", func(t *testing.T) {
			_, err := engines.CreateBlackboxGradingEngine(
				sbx,
				nil,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello.c",
			)

			tester.AssertCustomError(t, err, languages.ErrLanguageNotExist)
		})

		t.Run("Returns error when sandbox is Nil", func(t *testing.T) {
			_, err := engines.CreateBlackboxGradingEngine(
				nil,
				languages.CGradingLanguage,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello.c",
			)

			tester.AssertCustomError(t, err, sandboxes.ErrSandboxIsNil)
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("Return EngineRunResult", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 300)
			if err != nil {
				t.Fatal(err)
			}
			defer sbx.Cleanup()

			if err := sbx.AddFile("../../../tests/c_test/hello_binary/1.in"); err != nil {
				t.Fatal(err)
			}

			if err := sbx.AddFile("../../../tests/c_test/hello_binary/1.out"); err != nil {
				t.Fatal(err)
			}

			if err := sbx.AddFile("../../../tests/c_test/hello_binary/hello"); err != nil {
				t.Fatal(err)
			}

			os.Chmod(sbx.BoxDir+"/hello", 0700)

			engine, err := engines.CreateBlackboxGradingEngine(
				sbx,
				languages.CGradingLanguage,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello.c",
			)

			tester.AssertNotError(t, err)

			result, err := engine.Run("1.in", "1.out")

			tester.AssertNotError(t, err)
			if result.Verdict != models.VerdictAC {
				t.Errorf("expected Accepted status, instead got %+v", result)
			}
		})

		t.Run("Returns error when binary file is not found", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 302)
			if err != nil {
				t.Fatal(err)
			}
			defer sbx.Cleanup()

			if err := sbx.AddFile("../../../tests/c_test/hello/1.in"); err != nil {
				t.Fatal(err)
			}

			if err := sbx.AddFile("../../../tests/c_test/hello/1.out"); err != nil {
				t.Fatal(err)
			}

			engine, err := engines.CreateBlackboxGradingEngine(
				sbx,
				languages.CGradingLanguage,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello.c",
			)

			tester.AssertNotError(t, err)

			result, err := engine.Run("1.in", "1.out")

			tester.AssertNotError(t, err)
			if result.Verdict != models.VerdictRE || !strings.Contains(result.ErrorMessage, "No such file or directory") {
				t.Errorf("expected Runtime Error status with File not found error message, instead got %+v", result)
			}
		})
	})
}
