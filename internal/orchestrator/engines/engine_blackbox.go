package engines

import (
	"os"
	"path/filepath"

	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type BlackboxGradingEngine struct {
	Sandbox                    sandboxes.Sandbox
	LanguageOrBuilder          languages.Language
	Evaluator                  evaluator.Evaluator
	ExecutableScriptOrFilename string
}

func CreateBlackboxGradingEngine(sbx sandboxes.Sandbox, languageOrBuilder languages.Language, limits models.GradingLimit, evaluator evaluator.Evaluator, executableScriptOrFilename string) (*BlackboxGradingEngine, error) {
	if languageOrBuilder == nil {
		return nil, languages.ErrLanguageNotExist
	}

	if sbx == nil {
		return nil, sandboxes.ErrSandboxIsNil
	}

	sbx.SetTimeLimitInMiliseconds(limits.TimeInMiliseconds)
	sbx.SetWallTimeLimitInMiliseconds(limits.TimeInMiliseconds + 5000)
	sbx.SetMemoryLimitInKilobytes(limits.MemoryInKilobytes)

	return &BlackboxGradingEngine{
		Sandbox:                    sbx,
		LanguageOrBuilder:          languageOrBuilder,
		Evaluator:                  evaluator,
		ExecutableScriptOrFilename: executableScriptOrFilename,
	}, nil
}

func (ge BlackboxGradingEngine) Run(inputFilenameInBox, expectedOutputPath string, messageLimit int) (evaluator.EngineRunResult, error) {
	expectedOutputFilenameInBox := filepath.Base(expectedOutputPath)
	// Discard files a previous run may have replaced with symlinks before copying or redirecting.
	for _, filename := range []string{inputFilenameInBox, expectedOutputFilenameInBox + ".actual"} {
		if err := os.Remove(filepath.Join(ge.Sandbox.GetBoxdir(), filename)); err != nil && !os.IsNotExist(err) {
			return evaluator.EngineRunResult{Verdict: models.VerdictXX, HasErrorMessage: true, ErrorMessage: budgetedError(err, messageLimit)}, err
		}
	}
	if err := ge.Sandbox.AddFile(filepath.Join(filepath.Dir(expectedOutputPath), inputFilenameInBox)); err != nil {
		return evaluator.EngineRunResult{Verdict: models.VerdictXX, HasErrorMessage: true, ErrorMessage: budgetedError(err, messageLimit)}, err
	}
	// Do not leave an earlier test's hidden input available to later tests.
	defer os.Remove(filepath.Join(ge.Sandbox.GetBoxdir(), inputFilenameInBox))
	redirectionFiles := sandboxes.CreateRedirectionFiles(ge.Sandbox)

	if err := redirectionFiles.CreateNewMetaFileAndRedirect(expectedOutputFilenameInBox + ".meta"); err != nil {
		return evaluator.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	if err := redirectionFiles.RedirectStandardInput(inputFilenameInBox); err != nil {
		return evaluator.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	if err := redirectionFiles.CreateNewStandardOutputFileAndRedirect(expectedOutputFilenameInBox + ".actual"); err != nil {
		return evaluator.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	if err := redirectionFiles.RedirectStandardError(expectedOutputFilenameInBox + ".actual"); err != nil {
		return evaluator.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	ge.Sandbox.AddFileWithoutMove(expectedOutputFilenameInBox + ".actual")

	execResult := ge.Sandbox.Execute(
		ge.LanguageOrBuilder.GetExecutionCommand(ge.ExecutableScriptOrFilename),
		redirectionFiles,
	)

	thisRunResult := ge.Evaluator.Evaluate(ge.Sandbox, execResult, expectedOutputPath, expectedOutputFilenameInBox+".actual", messageLimit)
	thisRunResult.InputFilename = inputFilenameInBox
	thisRunResult.OutputFilename = expectedOutputFilenameInBox

	return thisRunResult, nil
}

func budgetedError(err error, messageLimit int) string {
	if messageLimit <= 0 {
		return ""
	}
	return diagnostics.Prefix(err.Error(), min(messageLimit, diagnostics.MaxBytes))
}
