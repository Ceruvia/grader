package compilers

const (
	CompilationOutputFilename = "_compile.out"
	CompilationMetaFilename   = "_compile.meta"
)

type Compiler interface {
	Compile(sourceFilenamesInsideBoxdir []string) (CompilerResult, error)
}
