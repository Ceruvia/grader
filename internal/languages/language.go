package languages

import (
	"errors"

	"github.com/Ceruvia/grader/internal/helper/command"
)

var (
	ErrLanguageNotExist = errors.New("Language or builder does not exist.")
)

type Language interface {
	GetName() string
	GetAllowedExtention() []string
	GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder
	GetExecutionCommand(binaryFilename string) command.CommandBuilder
	GetExecutableFilename(sourceFilename string) string
}
