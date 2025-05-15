package javalang

import (
	"github.com/Ceruvia/grader/internal/command"
	"github.com/Ceruvia/grader/internal/utils"
)

type JavaLanguage struct{}

func (l JavaLanguage) GetName() string {
	return "Java"
}

func (l JavaLanguage) GetAllowedExtention() []string {
	return []string{"java"}
}

func (l JavaLanguage) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	executableFilename := utils.RemoveExtention(mainSourceFilename)
	mainClassName := l.GetExecutableFilename(mainSourceFilename)

	compileSourceToClassCommand := *command.GetCommandBuilder("/usr/bin/javac").
		AddArgs(sourceFilenames...).
		AddArgs(mainSourceFilename).AddArgs(sourceFilenames...)

	compileClassToJarCommand := *command.GetCommandBuilder("/usr/bin/jar").
		AddArgs("cfe").
		AddArgs(executableFilename).
		AddArgs(mainClassName).
		AddArgs("*.class")

	return *command.GetCommandBuilder("/bin/bash").AddArgs("-c").
		AddArgs(compileSourceToClassCommand.Program).AddArgs(compileSourceToClassCommand.Args...).
		AddArgs("&&").
		AddArgs(compileClassToJarCommand.Program).AddArgs(compileClassToJarCommand.Args...)
}

func (l JavaLanguage) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/java").
		AddArgs("-jar").
		AddArgs(l.GetExecutableFilename(mainSourceFilename))
}

func (l JavaLanguage) GetExecutableFilename(sourceFilename string) string {
	return utils.RemoveExtention(sourceFilename) + ".jar"
}
