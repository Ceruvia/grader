package cpplang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type Cpp17Language struct{}

func (l Cpp17Language) GetName() string {
	return "C++17"
}

func (l Cpp17Language) GetAllowedExtention() []string {
	return []string{"cpp", "cc"}
}

func (l Cpp17Language) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/g++").
		AddArgs("-std=c++17").
		AddArgs("-o").
		AddArgs(l.GetExecutableFilename(mainSourceFilename)).
		AddArgs(sourceFilenames...).
		AddArgs("-O2").AddArgs("-lm")
}

func (l Cpp17Language) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + l.GetExecutableFilename(mainSourceFilename))
}

func (l Cpp17Language) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
