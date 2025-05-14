package engines_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/compilers"
	"github.com/Ceruvia/grader/internal/engines"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestConstructor(t *testing.T) {
	t.Run("it should return error when language provided does not exist", func(t *testing.T) {
		_, err := engines.CreateBlackboxGradingEngine(&isolate.IsolateSandbox{}, models.Submission{
			Language: "gaada bahasanya abangku",
		})

		if err != languages.ErrLanguageNotExists {
			t.Errorf("expected LanguageNotExists error but instead got %q", err)
		}
	})

	t.Run("it should be able to create a BlackBoxEngine", func(t *testing.T) {
		sbx := &isolate.IsolateSandbox{
			BoxDir: "../../tests/not_commited",
		}
		submission := models.Submission{
			Id:             "awjofi92",
			TempDir:        "/temp/fake",
			Language:       "c",
			BuildFiles:     []string{},
			SubmittedFiles: []string{"hello.c"},
			TCInputFiles:   []string{"1.in"},
			TCOutputFiles:  []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		compiler, err := compilers.PrepareSingleSourceFileCompiler(sbx, languages.CGradingLanguage)
		if err != nil {
			t.Fatal(err)
		}

		engine, err := engines.CreateBlackboxGradingEngine(sbx, submission)
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		want := engines.BlackboxGradingEngine{
			Sandbox:  sbx,
			Compiler: compiler,
			Language: languages.CGradingLanguage,
		}

		utils.AssertDeep(t, engine, want)
	})
}

// type Submission struct {
// 	Id             string
// 	TempDir        string
// 	Language       string
// 	BuildFiles     []string // files originating from problem statement
// 	SubmittedFiles []string // files originating from user upload / submit
// 	TCInputFiles   []string
// 	TCOutputFiles  []string
// 	Limits         GradingLimit
// }
