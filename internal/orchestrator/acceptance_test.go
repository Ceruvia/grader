package orchestrator_test

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

const (
	DEBUG = false

	ISOLATE_PATH                    = "/usr/local/bin/isolate"
	C_TEST_ID_PREFIX                = 900
	C_MAKEFILE_TEST_ID_PREFIX       = 910
	JAVA_TEST_ID_PREFIX             = 920
	JAVA_MAKEFILE_TEST_ID_PREFIX    = 930
	HASKELL_TEST_ID_PREFIX          = 940
	HASKELL_MAKEFILE_TEST_ID_PREFIX = 950
	DOTTED_NAME_TEST_ID_PREFIX      = 970
	CASE_IDENTITY_TEST_ID           = 980
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
			OriginalFileDir: "../../tests/c_test/e2e/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Compile Error", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Kotak Bola",
			Submisison:      createCSubmission("kotakbola.c", 20, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/scs_kotakbola",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Fix Tags",
			Submisison:      createCSubmission("fixTags.c", 10, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/scs_adt_fixtags",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Hanoi",
			Submisison:      createCSubmission("hanoi.c", 10, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/scs_adt_hanoi",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Compile Error_Empty file",
			Submisison:      createCSubmission("empty.c", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_empty",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_No file",
			Submisison:      createCSubmission("gaada.c", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_nofile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_Unfound function",
			Submisison:      createCSubmission("unfoundfunc.c", 0, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_unfoundfunc",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "printf", []string{}),
		},
		{
			Title:           "Runtime Error_Null pointer",
			Submisison:      createCSubmission("nullpointer.c", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/re_nullpointer",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"RE", "AC"}),
		},
		{
			Title:           "Runtime Error_Divide by zero",
			Submisison:      createCSubmission("divide.c", 1, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/re_dividebyzero",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"RE"}),
		},
		{
			Title:           "Time Limit",
			Submisison:      createCSubmission("infiniteloop.c", 1, 200, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/tle_timelimit",
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

			if err := prepareSubmission(sbx, test.Submisison, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

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
			OriginalFileDir: "../../tests/c_test/e2e/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Compile Error", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Kotak Bola",
			Submisison:      createCWithMakefileSubmission("compile", "prog", 20, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/scs_kotakbola",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Fix Tags",
			Submisison:      createCWithMakefileSubmission("compile", "prog", 10, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/scs_adt_fixtags",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Compile Error_Empty file",
			Submisison:      createCWithMakefileSubmission("", "empty", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_empty",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "undefined reference to", []string{}),
		},
		{
			Title:           "Compile Error_No file",
			Submisison:      createCWithMakefileSubmission("", "hello", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_nofile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "empty.c: No such file or directory", []string{}),
		},
		{
			Title:           "Compile Error_No makefile",
			Submisison:      createCWithMakefileSubmission("", "hello", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/ce_nomakefile",
			ExpectedResult:  createExpectedResult(false, "Compile Error", "make: *** No targets specified and no makefile found.  Stop.", []string{}),
		},
		{
			Title:           "Runtime Error_Null pointer",
			Submisison:      createCWithMakefileSubmission("", "nullpointer", 2, 1000, 1024),
			OriginalFileDir: "../../tests/c_test/e2e/re_nullpointer",
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

			if err := prepareSubmission(sbx, test.Submisison, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

// A dotted source basename or run target names the artifact exactly once:
// my.sol.c builds and runs my.sol, and a makefile target program.out runs as is.
func TestGradingDottedExecutableNames(t *testing.T) {
	limits := createLimits(1000, 65536)
	GradingTests := []struct {
		Title      string
		Submisison models.Submission
	}{
		{"C source my.sol.c", models.SubmissionWithFiles{Core: models.Core{Language: "C", Limits: limits, Testcases: createTestcases(1)}, MainSourceFilename: "my.sol.c"}},
		{"C++17 source my.sol.cpp", models.SubmissionWithFiles{Core: models.Core{Language: "C++17", Limits: limits, Testcases: createTestcases(1)}, MainSourceFilename: "my.sol.cpp"}},
		{"Makefile target program.out", models.SubmissionWithBuilder{Core: models.Core{Language: "C", Limits: limits, Testcases: createTestcases(1)}, Builder: "Makefile", RunScript: "program.out"}},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, DOTTED_NAME_TEST_ID_PREFIX+i)
			if err != nil {
				t.Fatal(err)
			}
			defer sbx.Cleanup()

			if err := prepareSubmission(sbx, test.Submisison, "../../tests/c_test/e2e/scs_dotted"); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, createExpectedResult(true, "Success", "", []string{"AC"}))
		})
	}
}

// A case the grader cannot set up still reports which case it is, so the
// Backend can store it as that case's internal error.
func TestGradingCaseSetupFailureKeepsIdentity(t *testing.T) {
	sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, CASE_IDENTITY_TEST_ID)
	if err != nil {
		t.Fatal(err)
	}
	defer sbx.Cleanup()
	submission := models.SubmissionWithFiles{
		Core:               models.Core{Language: "C", Limits: createLimits(1000, 65536), Testcases: createTestcases(2)},
		MainSourceFilename: "nextdir.c",
	}
	if err := prepareSubmission(sbx, submission, "../../tests/c_test/e2e/xx_next_input_dir"); err != nil {
		t.Fatal(err)
	}

	res := orchestrator.GradeBlackboxSubmission(sbx, submission)

	assertGradingResult(t, res, createExpectedResult(true, "Success", "", []string{"AC", "XX"}))
	for i, got := range res.TestcaseGradingResult {
		wantIn, wantOut := fmt.Sprintf("%d.in", i+1), fmt.Sprintf("%d.out", i+1)
		if got.InputFilename != wantIn || got.OutputFilename != wantOut {
			t.Errorf("case %d: want identity %s/%s, got %+v", i+1, wantIn, wantOut, got)
		}
	}
	// Staff see this message, so it must not reveal grader host paths.
	xx := res.TestcaseGradingResult[1]
	if !xx.HasErrorMessage || xx.ErrorMessage == "" {
		t.Errorf("case 2 must carry an error message, got %+v", xx)
	}
	for _, hostPath := range []string{"/var/", "/tmp/", sbx.BoxDir} {
		if strings.Contains(xx.ErrorMessage, hostPath) {
			t.Errorf("case 2 message reveals host path %q: %q", hostPath, xx.ErrorMessage)
		}
	}
}

func TestGradingJava(t *testing.T) {
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("no JDK in grader image")
	}
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
			OriginalFileDir: "../../tests/java_test/scs_hello",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "WA"}),
		},
		{
			Title:           "Success_Balala",
			Submisison:      createJavaSubmission("Main.java", 20, 1000, 1024),
			OriginalFileDir: "../../tests/java_test/scs_balala",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "WA", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "WA", "WA", "AC", "WA", "AC", "WA", "AC", "AC", "AC", "WA"}),
		},
		{
			Title:           "Success_Ngabuburit",
			Submisison:      createJavaSubmission("Main.java", 25, 1000, 1024),
			OriginalFileDir: "../../tests/java_test/scs_mult_ngabuburit",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Concurrency",
			Submisison:      createJavaSubmission("Main.java", 5, 1000, 1024),
			OriginalFileDir: "../../tests/java_test/srs_concurrency",
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

			if err := prepareSubmission(sbx, test.Submisison, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func TestGradingJavaWithMakefile(t *testing.T) {
	if _, err := exec.LookPath("javac"); err != nil {
		t.Skip("no JDK in grader image")
	}
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
			OriginalFileDir: "../../tests/java_test/scs_balala",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "WA", "AC", "AC", "AC", "WA", "AC", "AC", "AC", "WA", "WA", "AC", "WA", "AC", "WA", "AC", "AC", "AC", "WA"}),
		},
		{
			Title:           "Success_Ngabuburit",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 25, 1000, 1024),
			OriginalFileDir: "../../tests/java_test/scs_mult_ngabuburit",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Concurrency",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 5, 1000, 1024),
			OriginalFileDir: "../../tests/java_test/srs_concurrency",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "WA", "AC"}),
		},
		{
			Title:           "Success",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 5, 1000, 102400),
			OriginalFileDir: "../../tests/makefile_test/java_makefile",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Praktikum/InventoryManagement",
			Submisison:      createJavaWithMakefileSubmission("Main.class", "Main", 5, 1000, 102400),
			OriginalFileDir: "../../tests/java_test/scs_inventory_management",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, JAVA_MAKEFILE_TEST_ID_PREFIX+i)
			log.Println(test.Title, sbx.BoxDir)

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				// defer sbx.Cleanup()
			}

			if err := prepareSubmission(sbx, test.Submisison, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

			fmt.Printf("%+v\n", res)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func TestGradingHaskellWithMakefile(t *testing.T) {
	createHaskellWithMakefileSubmission := func(compileScript, runScript string, numOfTestcase, timeInMilisecond, memoryInKilobyte int) models.Submission {
		return models.SubmissionWithBuilder{
			Core: models.Core{
				Language:  "Haskell",
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
			Title:           "Success_Hitung Bensin",
			Submisison:      createHaskellWithMakefileSubmission("haskell", "Main", 10, 1000, 65536),
			OriginalFileDir: "../../tests/haskell_test/hitung_bensin",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC", "AC"}),
		},
		{
			Title:           "Success_Jam Bangun",
			Submisison:      createHaskellWithMakefileSubmission("haskell", "Main", 10, 1000, 65536),
			OriginalFileDir: "../../tests/haskell_test/jam_bangun",
			ExpectedResult:  createExpectedResult(true, "Success", "", []string{"AC", "AC", "WA", "WA", "WA", "WA", "AC", "WA", "AC", "AC"}),
		},
	}

	for i, test := range GradingTests {
		t.Run(test.Title, func(t *testing.T) {
			t.Parallel()
			sbx, err := sandboxes.CreateIsolateSandbox(ISOLATE_PATH, HASKELL_MAKEFILE_TEST_ID_PREFIX+i)

			sbx.AddAllowedDirectory("/home/abil")

			if err != nil {
				tester.AssertNotError(t, err)
			}
			if DEBUG {
				fmt.Println(sbx.BoxDir)
			} else {
				defer sbx.Cleanup()
			}

			if err := prepareSubmission(sbx, test.Submisison, test.OriginalFileDir); err != nil {
				t.Fatal(err)
			}

			res := orchestrator.GradeBlackboxSubmission(sbx, test.Submisison)

			assertGradingResult(t, res, test.ExpectedResult)
		})
	}
}

func assertGradingResult(t testing.TB, got, want evaluator.GradingResult) {
	t.Helper()

	if got.IsSuccess != want.IsSuccess {
		t.Fatalf("expected IsSuccess to be %t, instead got %+v", want.IsSuccess, got)
	}
	if !want.IsSuccess && got.Status != want.Status {
		t.Errorf("expected status %q, instead got %+v", want.Status, got)
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

func prepareSubmission(sandbox sandboxes.Sandbox, submission models.Submission, srcDir string) error {
	absDir, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}
	testcases := submission.GetTestcases()
	for i := range testcases {
		testcases[i].ExpectedOutputPath = filepath.Join(absDir, testcases[i].OutputFilename)
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) != ".out" {
			srcPath := filepath.Join(srcDir, entry.Name())
			if err := sandbox.AddFile(srcPath); err != nil {
				return err
			}
		}
	}

	return nil
}
