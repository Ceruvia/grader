package compilers

const (
	CompilationOutputFilename       = "_compile.out"
	CompilationMetaFilename         = "_compile.meta"
	CompilationBinaryOutputFilename = "outfile"
)

type Compiler interface {
	Compile(mainSourceFilename string, sourceFilenamesInsideBoxdir []string) (CompilerResult, error)
}
