package engines_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/engines"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
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
				"hello",
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
				"hello",
			)

			tester.AssertCustomError(t, err, languages.ErrLanguageNotExist)
		})

		t.Run("Returns error when sandbox is Nil", func(t *testing.T) {
			_, err := engines.CreateBlackboxGradingEngine(
				nil,
				languages.CGradingLanguage,
				Limit,
				evaluator.SimpleEvaluator{},
				"hello",
			)

			tester.AssertCustomError(t, err, sandboxes.ErrSandboxIsNil)
		})
	})
}
