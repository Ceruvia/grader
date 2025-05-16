package pylang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
)

type Python3Language struct{}

func (l Python3Language) GetName() string {
	return "Python 3"
}

func (l Python3Language) GetAllowedExtention() []string {
	return []string{"py"}
}

func (l Python3Language) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/bin/true")
}

func (l Python3Language) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/python3").
		AddArgs(mainSourceFilename)
}

func (l Python3Language) GetExecutableFilename(sourceFilename string) string {
	return sourceFilename
}
