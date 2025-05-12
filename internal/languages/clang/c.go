package clang

import "github.com/Ceruvia/grader/internal/command"

type CLanguage struct{}

func (l CLanguage) GetName() string {
	return "c"
}

func (l CLanguage) GetAllowedExtention() []string {
	return []string{"c"}
}

func (l CLanguage) GetCompilationCommand(binaryFilename string, buildFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("-std=gnu99").AddArgs("-o").AddArgs(binaryFilename).AddArgs(buildFilenames...).AddArgs("-O2").AddArgs("lm")
}

func (l CLanguage) GetExecutionCommand(binaryFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + binaryFilename)
}
