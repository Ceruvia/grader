package builder

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type MakefileBuilder struct{}

func (b MakefileBuilder) GetName() string {
	return "Makefile"
}

func (b MakefileBuilder) GetAllowedExtention() []string {
	return []string{}
}

func (b MakefileBuilder) GetCompilationCommand(compileCommand string, _ ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/make").
		AddArgs(compileCommand)
}

func (b MakefileBuilder) GetExecutionCommand(binaryFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + binaryFilename)
}

func (b MakefileBuilder) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
