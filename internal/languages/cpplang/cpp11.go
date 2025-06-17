package cpplang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type Cpp11Language struct{}

func (l Cpp11Language) GetName() string {
	return "C++11"
}

func (l Cpp11Language) GetAllowedExtention() []string {
	return []string{"cpp", "cc"}
}

func (l Cpp11Language) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/g++").
		AddArgs("-std=c++11").
		AddArgs("-o").
		AddArgs(l.GetExecutableFilename(mainSourceFilename)).
		AddArgs(sourceFilenames...).
		AddArgs("-O2").AddArgs("-lm")
}

func (l Cpp11Language) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + l.GetExecutableFilename(mainSourceFilename))
}

func (l Cpp11Language) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
