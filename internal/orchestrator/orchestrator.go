package orchestrator

import (
	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	"github.com/Ceruvia/grader/internal/orchestrator/engines"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
	log "github.com/sirupsen/logrus"
)

const StatusInternalError = "Internal error"

func GradeBlackboxSubmission(sandbox sandboxes.Sandbox, submission models.Submission) evaluator.GradingResult {
	if submission.GetLanguage() == nil {
		return createFailGradingResult("Compile Error", languages.ErrLanguageNotExist.Error())
	}

	compileResult, err := compilers.CompileSubmission(sandbox, submission, diagnostics.MaxBytes)
	if err != nil {
		return createFailGradingResult(StatusInternalError, err.Error())
	}
	if compileResult.InternalError {
		return createFailGradingResult(StatusInternalError, compileResult.StdoutStderr)
	}

	// Instantly return failed GradingResult if compilation failed
	if !compileResult.IsSuccess {
		return createFailGradingResult("Compile Error", compileResult.StdoutStderr)
	}

	/* Run against testcases and grade */
	/* Create language, compiler and engine */
	engine, err := engines.CreateBlackboxGradingEngine(
		sandbox,
		submission.GetLanguage(),
		submission.GetLimits(),
		evaluator.SimpleEvaluator{},
		submission.GetExecFilenameOrScript(),
	)
	if err != nil {
		return createFailGradingResult(StatusInternalError, err.Error())
	}

	testcases := submission.GetTestcases()
	gradingResults := make([]evaluator.EngineRunResult, 0, len(testcases))
	remaining := diagnostics.MaxBytes
	for _, tc := range testcases {
		// A case that could not run is already XX in runResult. Log the cause for staff.
		runResult, err := engine.Run(tc.InputFilename, tc.ExpectedOutputPath, remaining)
		if err != nil {
			log.WithFields(log.Fields{"submissionId": submission.GetId(), "testcase": tc.InputFilename}).WithError(err).Warn("Testcase could not run, recorded as XX")
		}
		remaining -= len(runResult.ErrorMessage)
		gradingResults = append(gradingResults, runResult)
	}

	return evaluator.GradingResult{
		Status:                "Success",
		IsSuccess:             true,
		TestcaseGradingResult: gradingResults,
	}
}
