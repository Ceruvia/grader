package clang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type CLanguage struct{}

func (l CLanguage) GetName() string {
	return "C"
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
	return files.RemoveExtention(sourceFilename)
}
