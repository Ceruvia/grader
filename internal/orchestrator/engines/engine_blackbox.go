package engines

import (
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
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
	sbx.SetWallTimeLimitInMiliseconds(limits.TimeInMiliseconds)
	sbx.SetMemoryLimitInKilobytes(limits.MemoryInKilobytes)

	return &BlackboxGradingEngine{
		Sandbox:                    sbx,
		LanguageOrBuilder:          languageOrBuilder,
		Evaluator:                  evaluator,
		ExecutableScriptOrFilename: languageOrBuilder.GetExecutableFilename(executableScriptOrFilename),
	}, nil
}

func (ge BlackboxGradingEngine) Run(inputFilenameInBox, expectedOutputFilenameInBox string) (evaluator.EngineRunResult, error) {
	redirectionFiles := sandboxes.CreateRedirectionFiles(ge.Sandbox.GetBoxdir())

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

	ge.Sandbox.AddFileWithoutMove(expectedOutputFilenameInBox + ".meta")
	ge.Sandbox.AddFileWithoutMove(expectedOutputFilenameInBox + ".actual")

	execResult := ge.Sandbox.Execute(
		ge.LanguageOrBuilder.GetExecutionCommand(ge.ExecutableScriptOrFilename),
		redirectionFiles,
	)

	thisRunResult := ge.Evaluator.Evaluate(ge.Sandbox, execResult, expectedOutputFilenameInBox, expectedOutputFilenameInBox+".actual")
	thisRunResult.InputFilename = inputFilenameInBox
	thisRunResult.OutputFilename = expectedOutputFilenameInBox

	return thisRunResult, nil
}
