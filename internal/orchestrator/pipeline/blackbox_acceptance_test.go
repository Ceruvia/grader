package pipeline_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/orchestrator/pipeline"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

const (
	ISOLATE_PATH     = "/usr/local/bin/isolate"
	C_TEST_ID_PREFIX = 900
)

func TestGradingC(t *testing.T) {
	createCSubmission := func(mainSourceFilename string, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithFiles{
			Core: models.Core{
				Language:  "C",
				Limits:    createLimits(timeInMilisecond, memoryInKilobyte),
				Testcases: createTestcases(2),
			},
			MainSourceFilename: mainSourceFilename,
		}
	}

	GradingTests := []struct {
		Title           string
		Submisison      models.Submission
		OriginalFileDir string
		ExpectedResult  evaluator.GradingResult
	}{
		{
			Title:           "Hello world",
			Submisison:      createCSubmission("hello.c", 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/hello",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "WA"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, C_TEST_ID_PREFIX+i)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			defer sbx.Cleanup()

			if err := moveToSandbox(sbx, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := pipeline.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func assertGradingResult(t testing.TB, got, want evaluator.GradingResult) {
	t.Helper()

	if got.IsSuccess != want.IsSuccess {
		t.Errorf("expected IsSuccess to be %t, instead got %+v", want.IsSuccess, got)
	}

	if len(got.TestcaseGradingResult) != len(want.TestcaseGradingResult) {
		t.Fatalf("expected got (%d) and want (%d) results to be the same number, instead got %+v", len(got.TestcaseGradingResult), len(want.TestcaseGradingResult), got)
	}

	for i, _ := range got.TestcaseGradingResult {
		if got.TestcaseGradingResult[i].Verdict != want.TestcaseGradingResult[i].Verdict {
			t.Errorf("expected %q TC to have %s verdict, instead got %+v", want.TestcaseGradingResult[i].InputFilename+"-"+want.TestcaseGradingResult[i].OutputFilename, want.TestcaseGradingResult[i].Verdict, got.TestcaseGradingResult[i])
		}
	}

}

func createExpectedResult(isSuccess bool, status, errorMessage string, verdicts []string) evaluator.GradingResult {
	var verdictToEngineRunResult []evaluator.EngineRunResult
	for i, verdict := range verdicts {
		var v models.Verdict
		switch verdict {
		case "AC":
			v = models.VerdictAC
		case "RE":
			v = models.VerdictRE
		case "WA":
			v = models.VerdictWA
		case "CE":
			v = models.VerdictCE
		case "TLE":
			v = models.VerdictTLE
		default:
			v = models.VerdictXX
		}

		verdictToEngineRunResult = append(verdictToEngineRunResult, evaluator.EngineRunResult{
			Verdict:        v,
			InputFilename:  fmt.Sprintf("%d.in", i+1),
			OutputFilename: fmt.Sprintf("%d.out", i+1),
		})
	}

	return evaluator.GradingResult{
		IsSuccess:             isSuccess,
		Status:                status,
		ErrorMessage:          errorMessage,
		TestcaseGradingResult: verdictToEngineRunResult,
	}
}

func createLimits(timeInMilisecond, memoryInKilobyte int) models.GradingLimit {
	return models.GradingLimit{
		TimeInMiliseconds: timeInMilisecond,
		MemoryInKilobytes: memoryInKilobyte,
	}
}

func createTestcases(numOfTestcase int) []models.Testcase {
	var tc []models.Testcase
	for i := range numOfTestcase {
		tc = append(tc, models.Testcase{
			InputFilename:  fmt.Sprintf("%d.in", i+1),
			OutputFilename: fmt.Sprintf("%d.out", i+1),
		})
	}
	return tc
}

func moveToSandbox(sandbox sandboxes.Sandbox, srcDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() {
			srcPath := filepath.Join(srcDir, entry.Name())
			if err := sandbox.AddFile(srcPath); err != nil {
				return err
			}
		}
	}

	return nil
}
