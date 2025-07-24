package haskelllang

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type HaskellLanguage struct{}

func (l HaskellLanguage) GetName() string {
	return "Haskell"
}

func (l HaskellLanguage) GetAllowedExtention() []string {
	return []string{"hs"}
}

func (l HaskellLanguage) GetCompilationCommand(mainSourceFilename string, sourceFilenames ...string) command.CommandBuilder {
	return *command.GetCommandBuilder("/usr/bin/ghc").
		AddArgs("--make").
		AddArgs(l.GetExecutableFilename(mainSourceFilename))
}

func (l HaskellLanguage) GetExecutionCommand(mainSourceFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + l.GetExecutableFilename(mainSourceFilename))
}

func (l HaskellLanguage) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
