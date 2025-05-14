package compilers

const (
	CompilationOutputFilename       = "_compile.out"
	CompilationMetaFilename         = "_compile.meta"
	CompilationBinaryOutputFilename = "outfile"
)

type Compiler interface {
	Compile(sourceFilenamesInsideBoxdir []string) (CompilerResult, error)
}
