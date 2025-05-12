package languages

import "github.com/Ceruvia/grader/internal/command"

type Language interface {
	GetName() string
	GetAllowedExtention() []string
	GetCompilationCommand(binaryFilename string, buildFilenames ...string) command.CommandBuilder
	GetExecutionCommand(binaryFilename string) command.CommandBuilder
}
