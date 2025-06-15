package cpplang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type Cpp20Language struct{}

func (l Cpp20Language) GetName() string {
	return "C++20"
}

func (l Cpp20Language) GetAllowedExtention() []string {
	return []string{"cpp", "cc"}
}

func (l Cpp20Language) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/g++").
		AddArgs("-std=c++20").
		AddArgs("-o").
		AddArgs(l.GetExecutableFilename(mainSourceFilename)).
		AddArgs(sourceFilenames...).
		AddArgs("-O2").AddArgs("-lm")
}

func (l Cpp20Language) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + l.GetExecutableFilename(mainSourceFilename))
}

func (l Cpp20Language) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
