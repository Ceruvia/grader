package languages

import "github.com/Ceruvia/grader/internal/command"

type Language interface {
	GetName() string
	GetAllowedExtention() []string
	GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder
	GetExecutionCommand(binaryFilename string) command.CommandBuilder
	GetExecutableFilename(sourceFilename string) string
}
