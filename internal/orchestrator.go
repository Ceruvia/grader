package internal

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Ceruvia/grader/internal/compilers"
	"github.com/Ceruvia/grader/internal/engines"
	"github.com/Ceruvia/grader/internal/evaluator"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
)

func GradeSubmission(boxId int, submission models.Submission) (models.GradingResult, error) {
	// 0. Get language simpleton
	language := languages.GetLanguageSimpleton(submission.Language)

	// 1. Create sandbox environment
	sandbox, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", boxId)
	if err != nil {
		return models.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, err
	}
	defer sandbox.Cleanup()

	// 2. Initialize boxdir with files
	err = moveToSandbox(&sandbox, submission.TempDir)
	if err != nil {
		return models.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, err
	}

	// 2.5 Get files with language extention
	sourceFilenames := []string{}
	for _, filename := range sandbox.GetFilenamesInBox() {
		for _, extention := range language.GetAllowedExtention() {
			if strings.HasSuffix(filename, "."+extention) {
				sourceFilenames = append(sourceFilenames, filename)
			}
		}
	}

	// 3. Compile file
	compiler, err := compilers.CreateCompilerBasedOnLang(&sandbox, language)
	if err != nil {
		return models.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, err
	}

	compilationRes, err := compiler.Compile(submission.MainSourceFilename, sourceFilenames)
	if !compilationRes.IsSuccess {
		return models.GradingResult{
			Status:       "Compile Error",
			IsSuccess:    false,
			ErrorMessage: compilationRes.StdoutStderr,
		}, err
	}
	if err != nil {
		return models.GradingResult{
			Status:       "Compile Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, err
	}

	// 4. Initialize engine
	engine, err := engines.CreateBlackboxGradingEngine(&sandbox, submission, evaluator.SimpleEvaluator{})
	if err != nil {
		return models.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, err
	}

	// 5. Grade all files
	var runResults []models.EngineRunResult

	for i, _ := range submission.TCInputFiles {
		runResult, _ := engine.Run(submission.TCInputFiles[i], submission.TCOutputFiles[i])
		runResults = append(runResults, runResult)
	}

	return models.GradingResult{
		Status:                "Success",
		IsSuccess:             true,
		TestcaseGradingResult: runResults,
	}, nil
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
