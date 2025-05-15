package internal

import (
	"fmt"
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

func GradeSubmission(boxId int, submission models.Submission) (models.EngineRunResult, error) {
	// 0. Get language simpleton
	language := languages.GetLanguageSimpleton(submission.Language)

	// 1. Create sandbox environment
	sandbox, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 990)
	if err != nil {
		return models.EngineRunResult{}, err
	}
	defer sandbox.Cleanup()

	// 2. Initialize boxdir with files
	err = moveToSandbox(&sandbox, submission.TempDir)
	if err != nil {
		return models.EngineRunResult{}, err
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
		return models.EngineRunResult{}, err
	}

	compilationRes, err := compiler.Compile(sourceFilenames)
	if !compilationRes.IsSuccess || err != nil {
		return models.EngineRunResult{}, err // TODO: Should return compilation error
	}

	// 4. Initialize engine
	engine, err := engines.CreateBlackboxGradingEngine(&sandbox, submission, evaluator.SimpleEvaluator{})
	if err != nil {
		return models.EngineRunResult{}, err
	}

	// 5. Grade all files
	for i, _ := range submission.TCInputFiles {
		runResult, err := engine.Run(submission.TCInputFiles[i], submission.TCOutputFiles[i])
		fmt.Printf("Input: %s	Output: %s\n", submission.TCInputFiles[i], submission.TCOutputFiles[i])
		fmt.Printf("Result: %+v\n", runResult)
		if err != nil {
			return models.EngineRunResult{}, err
		}

	}

	return models.EngineRunResult{}, nil
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
