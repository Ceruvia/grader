package clang

import (
	"github.com/Ceruvia/grader/internal/command"
	"github.com/Ceruvia/grader/internal/utils"
)

type CLanguage struct{}

func (l CLanguage) GetName() string {
	return "c"
}

func (l CLanguage) GetAllowedExtention() []string {
	return []string{"c"}
}

func (l CLanguage) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/gcc").
		AddArgs("-std=gnu99").
		AddArgs("-o").
		AddArgs(l.GetExecutableFilename(mainSourceFilename)).
		AddArgs(sourceFilenames...).
		AddArgs("-O2").AddArgs("-lm")
}

func (l CLanguage) GetExecutionCommand(binaryFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + binaryFilename)
}

func (l CLanguage) GetExecutableFilename(sourceFilename string) string {
	return utils.RemoveExtention(sourceFilename)
}
