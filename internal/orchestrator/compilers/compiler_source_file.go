package compilers

import (
	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

// Compile limits are variables only so tests can lower them. Production never changes them.
var (
	CompileTimeLimitMs     = 20 * 1000 // 20 seconds CPU
	CompileWallTimeLimitMs = 60 * 1000 // 1 minute wall
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
		Redirections:      sandboxes.CreateRedirectionFiles(sandbox),
	}

	compiler.Sandbox.SetTimeLimitInMiliseconds(CompileTimeLimitMs)
	compiler.Sandbox.SetWallTimeLimitInMiliseconds(CompileWallTimeLimitMs)
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

	compiler.Sandbox.AddFileWithoutMove(CompilationOutputFilename)

	return &compiler, nil
}

// Compiles the source files inside boxdir. Files are assumed to be in boxdir, and will be checked trough sandbox.
func (c SourceFileCompiler) Compile(mainSourceFilename string, sourceFilenamesInsideBoxdir []string, messageLimitBytes int64) CompilerResult {
	if messageLimitBytes <= 0 || messageLimitBytes > diagnostics.MaxBytes {
		messageLimitBytes = diagnostics.MaxBytes
	}
	compileCommand := c.LanguageOrBuilder.GetCompilationCommand(mainSourceFilename, sourceFilenamesInsideBoxdir...)

	result := c.Sandbox.Execute(compileCommand, c.Redirections)

	if result.Status == sandboxes.ZERO_EXIT_CODE {
		return CompilerResult{
			IsSuccess:      true,
			BinaryFilename: c.LanguageOrBuilder.GetExecutableFilename(mainSourceFilename),
		}
	} else if result.Status == sandboxes.NONZERO_EXIT_CODE || result.Status == sandboxes.KILLED_ON_SIGNAL {
		data, err := sandboxes.ReadBoxFile(c.Sandbox, CompilationOutputFilename, messageLimitBytes)

		if err != nil {
			// A submission can remove or replace its own diagnostic output.
			// This is a compilation failure, not a grader infrastructure failure.
			return CompilerResult{StdoutStderr: diagnostics.Limit(err.Error())}
		}
		return CompilerResult{StdoutStderr: diagnostics.Limit(string(data))}
	} else {
		return CompilerResult{
			InternalError: result.Status == sandboxes.INTERNAL_ERROR || result.Status == sandboxes.PARSING_META_ERROR,
			StdoutStderr:  diagnostics.Limit(result.Message),
		}
	}
}

func (c SourceFileCompiler) GetSandbox() sandboxes.Sandbox               { return c.Sandbox }
func (c SourceFileCompiler) GetRedirections() sandboxes.RedirectionFiles { return c.Redirections }
