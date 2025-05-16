package engines_test

import (
	"os"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/engines"
	"github.com/Ceruvia/grader/internal/evaluator"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestConstructor(t *testing.T) {
	t.Run("it should return error when language provided does not exist", func(t *testing.T) {
		_, err := engines.CreateBlackboxGradingEngine(&isolate.IsolateSandbox{}, models.Submission{
			Language: "gaada bahasanya abangku",
		}, evaluator.SimpleEvaluator{})

		if err != languages.ErrLanguageNotExists {
			t.Errorf("expected LanguageNotExists error but instead got %q", err)
		}
	})

	t.Run("it should be able to create a BlackBoxEngine", func(t *testing.T) {
		sbx := &isolate.IsolateSandbox{
			BoxDir: "../../tests/not_commited",
		}
		submission := models.Submission{
			Id:            "awjofi92",
			TempDir:       "/temp/fake",
			Language:      "c",
			TCInputFiles:  []string{"1.in"},
			TCOutputFiles: []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		engine, err := engines.CreateBlackboxGradingEngine(sbx, submission, evaluator.SimpleEvaluator{})
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		want := engines.BlackboxGradingEngine{
			Sandbox:   sbx,
			Language:  languages.CGradingLanguage,
			Evaluator: evaluator.SimpleEvaluator{},
		}

		utils.AssertDeep(t, engine, want)
	})
}

func TestRun(t *testing.T) {
	submission := models.Submission{
		Id:                 "awjofi92",
		TempDir:            "../../tests/c_test/hello",
		Language:           "c",
		MainSourceFilename: "hello.c",
		TCInputFiles:       []string{"1.in"},
		TCOutputFiles:      []string{"1.out"},
		Limits: models.GradingLimit{
			TimeInMiliseconds: 1000,
			MemoryInKilobytes: 102400,
		},
	}
	t.Run("it should error when the binary file isn't found", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 15)
		if err != nil {
			t.Fatal(err)
		}
		defer sbx.Cleanup()

		err = sbx.AddFile(submission.TempDir + "/1.in")
		if err != nil {
			t.Fatal(err)
		}
		err = sbx.AddFile(submission.TempDir + "/1.out")
		if err != nil {
			t.Fatal(err)
		}

		engine, err := engines.CreateBlackboxGradingEngine(&sbx, submission, evaluator.SimpleEvaluator{})
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		result, err := engine.Run("1.in", "1.out")

		utils.AssertNotError(t, err)
		if result.Verdict != models.VerdictRE || !strings.Contains(result.ErrorMessage, "No such file or directory") {
			t.Errorf("expected Runtime Error status with File not found error message, instead got %+v", result)
		}
	})

	t.Run("it should be able to run with TC", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 990)
		if err != nil {
			t.Fatal(err)
		}
		defer sbx.Cleanup()

		err = sbx.AddFile(submission.TempDir + "/1.in")
		if err != nil {
			t.Fatal(err)
		}
		err = sbx.AddFile(submission.TempDir + "/1.out")
		if err != nil {
			t.Fatal(err)
		}
		err = sbx.AddFile(submission.TempDir + "/hello") // Give exec permission
		if err != nil {
			t.Fatal(err)
		}
		os.Chmod(sbx.BoxDir+"/hello", 0700)

		engine, err := engines.CreateBlackboxGradingEngine(&sbx, submission, evaluator.SimpleEvaluator{})
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		result, err := engine.Run("1.in", "1.out")

		utils.AssertNotError(t, err)
		if result.Verdict != models.VerdictAC {
			t.Errorf("expected Accepted status, instead got %+v", result)
		}
	})
}
