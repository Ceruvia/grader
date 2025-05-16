package javalang

import (
	"fmt"

	"github.com/Ceruvia/grader/internal/helper"
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type JavaLanguage struct{}

func (l JavaLanguage) GetName() string {
	return "Java"
}

func (l JavaLanguage) GetAllowedExtention() []string {
	return []string{"java"}
}

func (l JavaLanguage) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	executableFilename := l.GetExecutableFilename(mainSourceFilename)
	mainClassName := files.RemoveExtention(mainSourceFilename)

	javacCommand := *command.GetCommandBuilder("/usr/bin/javac").
		AddArgs(helper.Map(sourceFilenames, quote)...)

	jarCommand := *command.GetCommandBuilder("/usr/bin/jar").
		AddArgs("cfe").
		AddArgs(quote(executableFilename)).
		AddArgs(quote(mainClassName)).
		AddArgs("*.class")

	cmdStr := fmt.Sprintf("%s && %s", javacCommand.BuildFullCommand(), jarCommand.BuildFullCommand())

	return *command.GetCommandBuilder("/bin/bash").
		AddArgs("-c").
		AddArgs(cmdStr)
}

func (l JavaLanguage) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/java").
		AddArgs("-jar").
		AddArgs(l.GetExecutableFilename(mainSourceFilename))
}

func (l JavaLanguage) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename) + ".jar"
}

func quote(filename string) string {
	return `"` + filename + `"`
}
