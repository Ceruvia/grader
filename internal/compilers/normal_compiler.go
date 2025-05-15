package compilers

import (
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type SingleSourceFileCompiler struct {
	Sandbox      sandboxes.Sandbox
	Redirections sandboxes.RedirectionFiles
	Language     languages.Language
}

func PrepareSingleSourceFileCompiler(sandbox sandboxes.Sandbox, language languages.Language) (SingleSourceFileCompiler, error) {
	compiler := SingleSourceFileCompiler{
		Sandbox:      sandbox,
		Language:     language,
		Redirections: sandboxes.CreateRedirectionFiles(sandbox.GetBoxdir()),
	}

	compiler.Sandbox.SetTimeLimitInMiliseconds(20 * 1000)   // 20 seconds
	compiler.Sandbox.SetMemoryLimitInKilobytes(1024 * 1024) // 1 GB

	err := compiler.Redirections.CreateNewMetaFileAndRedirect(CompilationMetaFilename)
	if err != nil {
		return SingleSourceFileCompiler{}, err
	}

	err = compiler.Redirections.CreateNewStandardOutputFileAndRedirect(CompilationOutputFilename)
	if err != nil {
		return SingleSourceFileCompiler{}, err
	}

	err = compiler.Redirections.RedirectStandardError(CompilationOutputFilename)
	if err != nil {
		return SingleSourceFileCompiler{}, err
	}

	return compiler, nil
}

// Compiles the source files inside boxdir. Files are assumed to be in boxdir, and will be checked trough sandbox.
func (c SingleSourceFileCompiler) Compile(mainSourceFilename string, sourceFilenamesInsideBoxdir []string) (CompilerResult, error) {
	compileCommand := c.Language.GetCompilationCommand(mainSourceFilename, sourceFilenamesInsideBoxdir...)
	result, err := c.Sandbox.Execute(compileCommand, c.Redirections)
	if err != nil {
		return CompilerResult{
			IsSuccess:    false,
			StdoutStderr: err.Error(),
		}, err
	}

	if result.Status == sandboxes.ZERO_EXIT_CODE {
		return CompilerResult{
			IsSuccess:      true,
			BinaryFilename: c.Language.GetExecutableFilename(mainSourceFilename),
		}, nil
	} else if result.Status == sandboxes.NONZERO_EXIT_CODE {
		data, err := c.Sandbox.GetFile(CompilationOutputFilename)
		return CompilerResult{
			IsSuccess:    false,
			StdoutStderr: string(data),
		}, err
	} else {
		return CompilerResult{
			IsSuccess: false,
		}, nil
	}
}
