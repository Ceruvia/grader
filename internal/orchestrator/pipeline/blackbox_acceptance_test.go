package pipeline_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/orchestrator/pipeline"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

const (
	DEBUG = false

	ISOLATE_PATH                 = "/usr/local/bin/isolate"
	C_TEST_ID_PREFIX             = 900
	C_MAKEFILE_TEST_ID_PREFIX    = 910
	JAVA_TEST_ID_PREFIX          = 920
	JAVA_MAKEFILE_TEST_ID_PREFIX = 930
)

func TestGradingC(t *testing.T) {
	createCSubmission := func(mainSourceFilename string, numOfTestcase, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithFiles{
			Core: models.Core{
				Language:  "C",
				Limits:    createLimits(timeInMilisecond, memoryInKilobyte),
				Testcases: createTestcases(numOfTestcase),
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
			Title:           "Success_Hello World",
			Submisison:      createCSubmission("hello.c", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Compile Error", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Kotak Bola",
			Submisison:      createCSubmission("kotakbola.c", 20, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_kotakbola",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Fix Tags",
			Submisison:      createCSubmission("fixTags.c", 10, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_adt_fixtags",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Hanoi",
			Submisison:      createCSubmission("hanoi.c", 10, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_adt_hanoi",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Compile Error_Empty file",
			Submisison:      createCSubmission("empty.c", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_empty",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_No file",
			Submisison:      createCSubmission("gaada.c", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_nofile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_Unfound function",
			Submisison:      createCSubmission("unfoundfunc.c", 0, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_unfoundfunc",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "error: implicit declaration of function ‘prinf’;", []string{}),
		},
		{
			Title:           "Runtime Error_Null pointer",
			Submisison:      createCSubmission("nullpointer.c", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/re_nullpointer",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"RE", "AC"}),
		},
		{
			Title:           "Runtime Error_Divide by zero",
			Submisison:      createCSubmission("divide.c", 1, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/re_dividebyzero",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"RE"}),
		},
		{
			Title:           "Time Limit",
			Submisison:      createCSubmission("infiniteloop.c", 1, 200, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/tle_timelimit",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"TLE"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, C_TEST_ID_PREFIX+i)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				defer sbx.Cleanup()
			}

			if err := moveToSandbox(sbx, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := pipeline.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func TestGradingCWithMakefile(t *testing.T) {
	createCWithMakefileSubmission := func(compileScript, runScript string, numOfTestcase, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithBuilder{
			Core: models.Core{
				Language:  "C",
				Limits:    createLimits(timeInMilisecond, memoryInKilobyte),
				Testcases: createTestcases(numOfTestcase),
			},
			Builder:       "Makefile",
			RunScript:     runScript,
			CompileScript: compileScript,
		}
	}

	GradingTests := []struct {
		Title           string
		Submisison      models.Submission
		OriginalFileDir string
		ExpectedResult  evaluator.GradingResult
	}{
		{
			Title:           "Success_Hello World",
			Submisison:      createCWithMakefileSubmission("", "hello", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Compile Error", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Kotak Bola",
			Submisison:      createCWithMakefileSubmission("compile", "prog", 20, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_kotakbola",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Fix Tags",
			Submisison:      createCWithMakefileSubmission("compile", "prog", 10, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/scs_adt_fixtags",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Compile Error_Empty file",
			Submisison:      createCWithMakefileSubmission("", "empty", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_empty",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_No file",
			Submisison:      createCWithMakefileSubmission("", "hello", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_nofile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "empty.c: No such file or directory", []string{}),
		},
		{
			Title:           "Compile Error_No makefile",
			Submisison:      createCWithMakefileSubmission("", "hello", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/ce_nomakefile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "make: *** No targets specified and no makefile found.  Stop.", []string{}),
		},
		{
			Title:           "Runtime Error_Null pointer",
			Submisison:      createCWithMakefileSubmission("", "nullpointer", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/c_test/e2e/re_nullpointer",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"RE", "AC"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, C_MAKEFILE_TEST_ID_PREFIX+i)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				defer sbx.Cleanup()
			}

			if err := moveToSandbox(sbx, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := pipeline.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func TestGradingJava(t *testing.T) {
	createJavaSubmission := func(mainSourceFilename string, numOfTestcase, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithFiles{
			Core: models.Core{
				Language:  "Java",
				Limits:    createLimits(timeInMilisecond, memoryInKilobyte),
				Testcases: createTestcases(numOfTestcase),
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
			Title:           "Success_Hello World",
			Submisison:      createJavaSubmission("HelloWorld.java", 2, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Balala",
			Submisison:      createJavaSubmission("Main.java", 20, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/scs_balala",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "WA", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "WA", "WA", "AC", "WA", "AC", "WA", "AC", "AC", "AC", "WA"}),
		},
		{
			Title:           "Success_Ngabuburit",
			Submisison:      createJavaSubmission("Main.java", 25, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/scs_mult_ngabuburit",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Concurrency",
			Submisison:      createJavaSubmission("Main.java", 5, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/srs_concurrency",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "WA", "AC"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, JAVA_TEST_ID_PREFIX+i)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				defer sbx.Cleanup()
			}

			if err := moveToSandbox(sbx, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := pipeline.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func TestGradingJavaWithMakefile(t *testing.T) {
	createJavaWithMakefileSubmission := func(compileScript, runScript string, numOfTestcase, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithBuilder{
			Core: models.Core{
				Language:  "Java",
				Limits:    createLimits(timeInMilisecond, memoryInKilobyte),
				Testcases: createTestcases(numOfTestcase),
			},
			Builder:       "Makefile",
			CompileScript: compileScript,
			RunScript:     runScript,
		}
	}

	GradingTests := []struct {
		Title           string
		Submisison      models.Submission
		OriginalFileDir string
		ExpectedResult  evaluator.GradingResult
	}{
		{
			Title:           "Success_Balala",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 20, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/scs_balala",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "WA", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "WA", "WA", "AC", "WA", "AC", "WA", "AC", "AC", "AC", "WA"}),
		},
		{
			Title:           "Success_Ngabuburit",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 25, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/scs_mult_ngabuburit",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Concurrency",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 5, 1000, 1024),
			OriginalFileDir: "../../../tests/java_test/srs_concurrency",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "WA", "AC"}),
		},
		{
			Title:           "Success",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 5, 1000, 102400),
			OriginalFileDir: "../../../tests/makefile_test/java_makefile",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, JAVA_MAKEFILE_TEST_ID_PREFIX+i)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				defer sbx.Cleanup()
			}

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
		t.Fatalf("expected IsSuccess to be %t, instead got %+v", want.IsSuccess, got)
	}

	if !strings.Contains(got.ErrorMessage, want.ErrorMessage) {
		t.Errorf("expected to get %q errorMessage, instead got %+v", want.ErrorMessage, got)
	}

	if got.Status != "Compile Error" {
		if len(got.TestcaseGradingResult) != len(want.TestcaseGradingResult) {
			t.Fatalf("expected got (%d) and want (%d) results to be the same number, instead got %+v", len(got.TestcaseGradingResult), len(want.TestcaseGradingResult), got)
		}

		for i, _ := range got.TestcaseGradingResult {
			if got.TestcaseGradingResult[i].Verdict != want.TestcaseGradingResult[i].Verdict {
				t.Errorf("expected %q TC to have %s verdict, instead got %+v", want.TestcaseGradingResult[i].InputFilename+"-"+want.TestcaseGradingResult[i].OutputFilename, want.TestcaseGradingResult[i].Verdict, got.TestcaseGradingResult[i])
			}
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
