package compilers

type CompilerResult struct {
	IsSuccess      bool
	BinaryFilename string
	StdoutStderr   string
}
