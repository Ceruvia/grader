package languages

import (
	"errors"

	"github.com/Ceruvia/grader/internal/command"
)

var (
	ErrLanguageNotExists = errors.New("language does not exists / implemented!")
)

type LanguageNotExists struct{}

func (l LanguageNotExists) GetName() string {
	return "not exists"
}

func (l LanguageNotExists) GetAllowedExtention() []string {
	return []string{}
}

func (l LanguageNotExists) GetCompilationCommand(binaryFilename string, buildFilenames ...string) command.CommandBuilder {
	return command.CommandBuilder{}
}

func (l LanguageNotExists) GetExecutionCommand(binaryFilename string) command.CommandBuilder {
	return command.CommandBuilder{}
}
