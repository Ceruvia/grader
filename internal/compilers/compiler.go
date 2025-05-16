package compilers

import "github.com/Ceruvia/grader/internal/sandboxes"

const (
	CompilationOutputFilename       = "_compile.out"
	CompilationMetaFilename         = "_compile.meta"
	CompilationBinaryOutputFilename = "outfile"
)

type Compiler interface {
	GetSandbox() sandboxes.Sandbox
	GetRedirections() sandboxes.RedirectionFiles
	Compile(mainSourceFilename string, sourceFilenamesInsideBoxdir []string) (CompilerResult, error)
}
