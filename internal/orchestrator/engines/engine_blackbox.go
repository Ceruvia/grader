package engines

import (
	"os"
	"path/filepath"

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
		ExecutableScriptOrFilename: languageOrBuilder.GetExecutableFilename(executableScriptOrFilename),
	}, nil
}

func (ge BlackboxGradingEngine) Run(inputFilenameInBox, expectedOutputPath string) (evaluator.EngineRunResult, error) {
	expectedOutputFilenameInBox := filepath.Base(expectedOutputPath)
	// Discard files a previous run may have replaced with symlinks before copying or redirecting.
	for _, filename := range []string{inputFilenameInBox, expectedOutputFilenameInBox + ".actual"} {
		if err := os.Remove(filepath.Join(ge.Sandbox.GetBoxdir(), filename)); err != nil && !os.IsNotExist(err) {
			return evaluator.EngineRunResult{Verdict: models.VerdictXX, HasErrorMessage: true, ErrorMessage: err.Error()}, err
		}
	}
	if err := ge.Sandbox.AddFile(filepath.Join(filepath.Dir(expectedOutputPath), inputFilenameInBox)); err != nil {
		return evaluator.EngineRunResult{Verdict: models.VerdictXX, HasErrorMessage: true, ErrorMessage: err.Error()}, err
	}
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

	thisRunResult := ge.Evaluator.Evaluate(ge.Sandbox, execResult, expectedOutputPath, expectedOutputFilenameInBox+".actual")
	thisRunResult.InputFilename = inputFilenameInBox
	thisRunResult.OutputFilename = expectedOutputFilenameInBox

	return thisRunResult, nil
}
