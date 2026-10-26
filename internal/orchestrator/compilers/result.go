package compilers

type CompilerResult struct {
	IsSuccess      bool
	InternalError  bool
	BinaryFilename string
	StdoutStderr   string
}
