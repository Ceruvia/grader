package orchestrator

import (
	"strings"

	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	"github.com/Ceruvia/grader/internal/orchestrator/engines"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func GradeBlackboxSubmission(sandbox sandboxes.Sandbox, submission models.Submission) evaluator.GradingResult {
	if submission.GetLanguage() == nil {
		return createFailGradingResult("Compile Error", languages.ErrLanguageNotExist.Error())
	}

	/* Create language, compiler and engine */
	engine, err := engines.CreateBlackboxGradingEngine(
		sandbox,
		submission.GetLanguage(),
		submission.GetLimits(),
		evaluator.SimpleEvaluator{},
		submission.GetExecFilenameOrScript(),
	)
	if err != nil {
		return createFailGradingResult("Internal error", err.Error())
	}
	compiler, err := compilers.PrepareSourceFileCompiler(sandbox, submission.GetCompileLanguage())
	if err != nil {
		return createFailGradingResult("Internal error", err.Error())
	}

	/* Compile source files to binary */

	// Get source filenames inside of box
	sourceFilenames := []string{}
	if !submission.IsBuilder() { // If it's builder then just skip, unecessary
		for _, filename := range sandbox.GetFilenamesInBox() {
			for _, extention := range submission.GetLanguage().GetAllowedExtention() {
				if strings.HasSuffix(filename, "."+extention) {
					sourceFilenames = append(sourceFilenames, filename)
				}
			}
		}
	}

	compileResult := compiler.Compile(submission.GetCompileFilenameOrScript(), sourceFilenames)

	// Instantly return failed GradingResult if compilation failed
	if !compileResult.IsSuccess {
		return createFailGradingResult("Compile Error", compileResult.StdoutStderr)
	}

	/* Run against testcases and grade */
	var gradingResults []evaluator.EngineRunResult

	for _, tc := range submission.GetTestcases() {
		runResult, _ := engine.Run(tc.InputFilename, tc.OutputFilename)
		gradingResults = append(gradingResults, runResult)
	}

	return evaluator.GradingResult{
		Status:                "Success",
		IsSuccess:             true,
		TestcaseGradingResult: gradingResults,
	}
}
