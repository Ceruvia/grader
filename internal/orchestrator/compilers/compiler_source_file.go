package compilers

import (
	"fmt"
	"os"

	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type SourceFileCompiler struct {
	Sandbox           sandboxes.Sandbox
	Redirections      sandboxes.RedirectionFiles
	LanguageOrBuilder languages.Language
}

func PrepareSourceFileCompiler(sandbox sandboxes.Sandbox, languageOrBuilder languages.Language) (*SourceFileCompiler, error) {
	if languageOrBuilder == nil {
		return nil, languages.ErrLanguageNotExist
	}

	compiler := SourceFileCompiler{
		Sandbox:           sandbox,
		LanguageOrBuilder: languageOrBuilder,
		Redirections:      sandboxes.CreateRedirectionFiles(sandbox.GetBoxdir()),
	}

	compiler.Sandbox.SetTimeLimitInMiliseconds(20 * 1000)  // 20 seconds
	compiler.Sandbox.SetWallTimeLimitInMiliseconds(60 * 1000)  // 1 minute
	compiler.Sandbox.SetMemoryLimitInKilobytes(1024 * 1024) // 1 GB

	if err := compiler.Redirections.CreateNewMetaFileAndRedirect(CompilationMetaFilename); err != nil {
		return nil, err
	}

	if err := compiler.Redirections.CreateNewStandardOutputFileAndRedirect(CompilationOutputFilename); err != nil {
		return nil, err
	}

	if err := compiler.Redirections.RedirectStandardError(CompilationOutputFilename); err != nil {
		return nil, err
	}

	compiler.Sandbox.AddFileWithoutMove(CompilationMetaFilename)
	compiler.Sandbox.AddFileWithoutMove(CompilationOutputFilename)

	return &compiler, nil
}

// Compiles the source files inside boxdir. Files are assumed to be in boxdir, and will be checked trough sandbox.
func (c SourceFileCompiler) Compile(mainSourceFilename string, sourceFilenamesInsideBoxdir []string) CompilerResult {
	compileCommand := c.LanguageOrBuilder.GetCompilationCommand(mainSourceFilename, sourceFilenamesInsideBoxdir...)

	fmt.Printf("%+v\n", c.GetSandbox())

	result := c.Sandbox.Execute(compileCommand, c.Redirections)

	if result.Status == sandboxes.ZERO_EXIT_CODE {
		return CompilerResult{
			IsSuccess:      true,
			BinaryFilename: c.LanguageOrBuilder.GetExecutableFilename(mainSourceFilename),
		}
	} else if result.Status == sandboxes.NONZERO_EXIT_CODE || result.Status == sandboxes.KILLED_ON_SIGNAL {
		data, err := c.Sandbox.GetFile(CompilationOutputFilename)

		if err != nil && err != os.ErrNotExist {
			return CompilerResult{
				IsSuccess:    false,
				StdoutStderr: err.Error(),
			}
		} else {
			return CompilerResult{
				IsSuccess:    false,
				StdoutStderr: string(data),
			}
		}
	} else {
		return CompilerResult{
			IsSuccess:    false,
			StdoutStderr: result.Message,
		}
	}
}

func (c SourceFileCompiler) GetSandbox() sandboxes.Sandbox               { return c.Sandbox }
func (c SourceFileCompiler) GetRedirections() sandboxes.RedirectionFiles { return c.Redirections }
