package engines

import (
	"github.com/Ceruvia/grader/internal/compilers"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type BlackboxGradingEngine struct {
	Sandbox  sandboxes.Sandbox
	Compiler compilers.Compiler
	Language languages.Language
}

func CreateBlackboxGradingEngine(sbx sandboxes.Sandbox, sub models.Submission) (BlackboxGradingEngine, error) {
	language := languages.GetLanguageSimpleton(sub.Language)
	if language.GetName() == "not exists" {
		return BlackboxGradingEngine{}, languages.ErrLanguageNotExists
	}

	compiler, err := compilers.CreateCompilerBasedOnLang(sbx, language)
	if err != nil {
		return BlackboxGradingEngine{}, err
	}

	return BlackboxGradingEngine{
		Sandbox:  sbx,
		Compiler: compiler,
		Language: language,
	}, nil
}

func (ge BlackboxGradingEngine) Run(inputFilenameInBox, expectedOutputFilenameInBox string) (sandboxes.SandboxExecutionResult, error) {
	redirectionFiles := sandboxes.CreateRedirectionFiles(ge.Sandbox.GetBoxdir())
	err := redirectionFiles.CreateNewMetaFileAndRedirect(expectedOutputFilenameInBox + ".meta")
	if err != nil {
		return sandboxes.SandboxExecutionResult{}, err
	}
	err = redirectionFiles.RedirectStandardInput(inputFilenameInBox)
	if err != nil {
		return sandboxes.SandboxExecutionResult{}, err
	}
	err = redirectionFiles.CreateNewStandardOutputFileAndRedirect(expectedOutputFilenameInBox + ".actual")
	if err != nil {
		return sandboxes.SandboxExecutionResult{}, err
	}
	err = redirectionFiles.RedirectStandardError(expectedOutputFilenameInBox + ".actual")
	if err != nil {
		return sandboxes.SandboxExecutionResult{}, err
	}

	result, err := ge.Sandbox.Execute(
		ge.Language.GetExecutionCommand(compilers.CompilationBinaryOutputFilename),
		redirectionFiles,
	)
	if err != nil {
		return sandboxes.SandboxExecutionResult{}, err
	}
	return result, nil
}
