package builder

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
)

type MakefileBuilder struct{}

func (b MakefileBuilder) GetName() string {
	return "Makefile"
}

func (b MakefileBuilder) GetAllowedExtention() []string {
	return []string{}
}

// GetCompilationCommand reads makefiles[0] with -f when given, so no other
// makefile in the box can take precedence. -B makes the teacher recipe run even
// when a student file with a future mtime already has a target name.
func (b MakefileBuilder) GetCompilationCommand(compileCommand string, makefiles ...string) command.CommandBuilder {
	cmd := command.GetCommandBuilder("/usr/bin/make")
	if len(makefiles) > 0 && makefiles[0] != "" {
		cmd.AddArgs("-B", "-f", makefiles[0])
	}
	if compileCommand != "" {
		cmd.AddArgs(compileCommand)
	}
	return *cmd
}

func (b MakefileBuilder) GetExecutionCommand(binaryFilename string) command.CommandBuilder {
	return *command.GetCommandBuilder("./" + binaryFilename)
}

func (b MakefileBuilder) GetExecutableFilename(sourceFilename string) string {
	return files.RemoveExtention(sourceFilename)
}
