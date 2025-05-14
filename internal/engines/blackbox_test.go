package engines_test

import (
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/engines"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
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

		engine, err := engines.CreateBlackboxGradingEngine(sbx, submission)
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		want := engines.BlackboxGradingEngine{
			Sandbox:  sbx,
			Language: languages.CGradingLanguage,
		}

		utils.AssertDeep(t, engine, want)
	})
}

func TestRun(t *testing.T) {
	submission := models.Submission{
		Id:             "awjofi92",
		TempDir:        "../../tests/c_test/hello",
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
	t.Run("it should error when the binary file isn't found", func(t *testing.T) {
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

		engine, err := engines.CreateBlackboxGradingEngine(&sbx, submission)
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		result, err := engine.Run("1.in", "1.out")

		utils.AssertNotError(t, err)
		if result.Message != "Exited with error status 127" || result.Status != sandboxes.NONZERO_EXIT_CODE {
			t.Errorf("expected error code 127 (not found) with RE, instead got %+v", result)
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
		err = sbx.AddFile(submission.TempDir + "/outfile") // Give exec permission
		if err != nil {
			t.Fatal(err)
		}
		os.Chmod(sbx.BoxDir+"/outfile", 0700)

		engine, err := engines.CreateBlackboxGradingEngine(&sbx, submission)
		if err != nil {
			t.Errorf("expected to get no error, instead got %q", err)
		}

		result, err := engine.Run("1.in", "1.out")

		utils.AssertNotError(t, err)
		if result.Status != sandboxes.ZERO_EXIT_CODE {
			t.Errorf("expected Success status, instead got %+v", result)
		}

		data, err := sbx.GetFile("1.out.actual")
		if err != nil {
			t.Fatal(err)
		}

		if string(data) != "Hello, World!\n" {
			t.Errorf("expected Hello, World!\n, instead got %s", string(data))
		}
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
