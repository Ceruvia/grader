package orchestrator_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
)

func TestGradingC(t *testing.T) {
	t.Run("it should be able to compile, run, and grade a simple Hello, World code", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "awjofi92",
			TempDir:            "../../tests/c_test/hello",
			Language:           "C",
			MainSourceFilename: "hello.c",
			TCInputFiles:       []string{"1.in"},
			TCOutputFiles:      []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(7, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
				},
			},
		}

		assertGradingResult(t, result, want)
	})

	t.Run("it should be able to compile, run, and grade an ADT question", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "dhsai82d",
			TempDir:            "../../tests/c_test/adt",
			Language:           "C",
			MainSourceFilename: "ganjilgenap.c",
			TCInputFiles:       []string{"1.in", "2.in", "3.in", "4.in", "5.in", "6.in", "7.in", "8.in", "9.in", "10.in"},
			TCOutputFiles:      []string{"1.out", "2.out", "3.out", "4.out", "5.out", "6.out", "7.out", "8.out", "9.out", "10.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(8, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "2.in",
					OutputFilename:  "2.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "3.in",
					OutputFilename:  "3.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "4.in",
					OutputFilename:  "4.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "5.in",
					OutputFilename:  "5.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "6.in",
					OutputFilename:  "6.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "7.in",
					OutputFilename:  "7.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "8.in",
					OutputFilename:  "8.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "9.in",
					OutputFilename:  "9.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "10.in",
					OutputFilename:  "10.out",
				},
			},
		}

		assertGradingResult(t, result, want)

	})

	t.Run("it should return a compile error if file is uncompileable", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "awjofi92",
			TempDir:            "../../tests/c_test/uncompileable_singular",
			Language:           "C",
			MainSourceFilename: "unfoundfunc.c",
			TCInputFiles:       []string{"1.in"},
			TCOutputFiles:      []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(9, submission)

		want := models.GradingResult{
			Status:    "Compile Error",
			IsSuccess: false,
		}

		assertGradingResult(t, result, want)
	})

	t.Run("it should return a RE verdict in one of it verdicts if file has RE", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "awjofi92",
			TempDir:            "../../tests/c_test/runtimeerror_singular",
			Language:           "C",
			MainSourceFilename: "nullpointer.c",
			TCInputFiles:       []string{"1.in", "2.in"},
			TCOutputFiles:      []string{"1.out", "2.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(10, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
					Verdict:         models.VerdictRE,
					HasErrorMessage: true,
				},
				{
					InputFilename:   "2.in",
					OutputFilename:  "2.out",
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
				},
			},
		}

		assertGradingResult(t, result, want)
	})

	t.Run("it should return a TLE verdict in one of it verdicts if it exceeds time limit", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "awjofi92",
			TempDir:            "../../tests/c_test/timelimit_singular",
			Language:           "C",
			MainSourceFilename: "infiniteloop.c",
			TCInputFiles:       []string{"1.in"},
			TCOutputFiles:      []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(11, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
					Verdict:         models.VerdictTLE,
					HasErrorMessage: true,
				},
			},
		}

		assertGradingResult(t, result, want)
	})
}

func TestGradingJava(t *testing.T) {
	t.Run("it should be able to compile, run, and grade a simple Hello, World code", func(t *testing.T) {
		submission := models.Submission{
			Id:                 "awjofi92",
			TempDir:            "../../tests/java_test/hello",
			Language:           "Java",
			MainSourceFilename: "HelloWorld.java",
			TCInputFiles:       []string{"1.in", "2.in"},
			TCOutputFiles:      []string{"1.out", "2.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(12, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
				},
				{
					Verdict:         models.VerdictWA,
					HasErrorMessage: false,
					InputFilename:   "2.in",
					OutputFilename:  "2.out",
				},
			},
		}

		assertGradingResult(t, result, want)
	})
}

func TestMakefile(t *testing.T) {
	t.Run("it should be able to use Makefile with C language", func(t *testing.T) {
		submission := models.Submission{
			Id:            "awjofi92",
			TempDir:       "../../tests/makefile_test/c_makefile",
			Language:      "c",
			UseBuilder:    true,
			Builder:       "Makefile",
			CompileScript: "compile",
			RunScript:     "prog",
			TCInputFiles:  []string{"1.in", "2.in", "3.in", "4.in", "5.in", "6.in", "7.in", "8.in", "9.in", "10.in"},
			TCOutputFiles: []string{"1.out", "2.out", "3.out", "4.out", "5.out", "6.out", "7.out", "8.out", "9.out", "10.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(201, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "2.in",
					OutputFilename:  "2.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "3.in",
					OutputFilename:  "3.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "4.in",
					OutputFilename:  "4.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "5.in",
					OutputFilename:  "5.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "6.in",
					OutputFilename:  "6.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "7.in",
					OutputFilename:  "7.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "8.in",
					OutputFilename:  "8.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "9.in",
					OutputFilename:  "9.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "10.in",
					OutputFilename:  "10.out",
				},
			},
		}

		assertGradingResult(t, result, want)
	})

	t.Run("it should be able to use Makefile with C language", func(t *testing.T) {
		submission := models.Submission{
			Id:            "awjofi92",
			TempDir:       "../../tests/makefile_test/java_makefile",
			Language:      "Java",
			UseBuilder:    true,
			Builder:       "Makefile",
			CompileScript: "Main.class",
			RunScript:     "Main",
			TCInputFiles:  []string{"1.in", "2.in", "3.in", "4.in", "5.in"},
			TCOutputFiles: []string{"1.out", "2.out", "3.out", "4.out", "5.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		result, _ := orchestrator.GradeSubmission(202, submission)

		want := models.GradingResult{
			Status:    "Success",
			IsSuccess: true,
			TestcaseGradingResult: []models.EngineRunResult{
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "1.in",
					OutputFilename:  "1.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "2.in",
					OutputFilename:  "2.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "3.in",
					OutputFilename:  "3.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "4.in",
					OutputFilename:  "4.out",
				},
				{
					Verdict:         models.VerdictAC,
					HasErrorMessage: false,
					InputFilename:   "5.in",
					OutputFilename:  "5.out",
				},
			},
		}

		assertGradingResult(t, result, want)
	})
}

func assertGradingResult(t testing.TB, got, want models.GradingResult) {
	t.Helper()

	if got.IsSuccess != want.IsSuccess {
		t.Errorf("expected IsSuccess to be %t, instead got %+v", want.IsSuccess, got)
	}

	if len(got.TestcaseGradingResult) != len(want.TestcaseGradingResult) {
		t.Fatalf("expected got (%d) and want (%d) results to be the same number, instead got +%v", len(got.TestcaseGradingResult), len(want.TestcaseGradingResult), got)
	}

	for i, _ := range got.TestcaseGradingResult {
		if got.TestcaseGradingResult[i].Verdict != want.TestcaseGradingResult[i].Verdict {
			t.Errorf("expected %q TC to have %s verdict, instead got %+v", want.TestcaseGradingResult[i].InputFilename+"-"+want.TestcaseGradingResult[i].OutputFilename, want.TestcaseGradingResult[i].Verdict, got.TestcaseGradingResult[i])
		}

		if got.TestcaseGradingResult[i].ErrorMessage != want.TestcaseGradingResult[i].ErrorMessage {
			t.Errorf("expected %q TC to have ErrorMessage %q, instead got %q", want.TestcaseGradingResult[i].InputFilename+"-"+want.TestcaseGradingResult[i].OutputFilename, want.TestcaseGradingResult[i].ErrorMessage, got.TestcaseGradingResult[i].ErrorMessage)
		}
	}

}
