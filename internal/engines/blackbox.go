package engines

import (
	"github.com/Ceruvia/grader/internal/compilers"
	"github.com/Ceruvia/grader/internal/evaluator"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type BlackboxGradingEngine struct {
	Sandbox   sandboxes.Sandbox
	Language  languages.Language
	Evaluator evaluator.Evaluator
}

func CreateBlackboxGradingEngine(sbx sandboxes.Sandbox, sub models.Submission, evaluator evaluator.Evaluator) (BlackboxGradingEngine, error) {
	language := languages.GetLanguageSimpleton(sub.Language)
	if language.GetName() == "not exists" {
		return BlackboxGradingEngine{}, languages.ErrLanguageNotExists
	}

	sbx.SetTimeLimitInMiliseconds(sub.Limits.TimeInMiliseconds)
	sbx.SetWallTimeLimitInMiliseconds(sub.Limits.TimeInMiliseconds)
	sbx.SetMemoryLimitInKilobytes(sub.Limits.MemoryInKilobytes)

	return BlackboxGradingEngine{
		Sandbox:   sbx,
		Language:  language,
		Evaluator: evaluator,
	}, nil
}

func (ge BlackboxGradingEngine) Run(inputFilenameInBox, expectedOutputFilenameInBox string) (models.EngineRunResult, error) {
	redirectionFiles := sandboxes.CreateRedirectionFiles(ge.Sandbox.GetBoxdir())
	err := redirectionFiles.CreateNewMetaFileAndRedirect(expectedOutputFilenameInBox + ".meta")
	if err != nil {
		return models.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}
	err = redirectionFiles.RedirectStandardInput(inputFilenameInBox)
	if err != nil {
		return models.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}
	err = redirectionFiles.CreateNewStandardOutputFileAndRedirect(expectedOutputFilenameInBox + ".actual")
	if err != nil {
		return models.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}
	err = redirectionFiles.RedirectStandardError(expectedOutputFilenameInBox + ".actual")
	if err != nil {
		return models.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	execResult, err := ge.Sandbox.Execute(
		ge.Language.GetExecutionCommand(compilers.CompilationBinaryOutputFilename),
		redirectionFiles,
	)

	if err != nil {
		return models.EngineRunResult{
			InputFilename:  inputFilenameInBox,
			OutputFilename: expectedOutputFilenameInBox,
			Verdict:        models.VerdictXX,
		}, err
	}

	thisRunResult, err := ge.Evaluator.Evaluate(ge.Sandbox, execResult, expectedOutputFilenameInBox, expectedOutputFilenameInBox+".actual")
	thisRunResult.InputFilename = inputFilenameInBox
	thisRunResult.OutputFilename = expectedOutputFilenameInBox

	return thisRunResult, err
}
