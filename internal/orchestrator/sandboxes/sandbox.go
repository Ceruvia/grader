package sandboxes

import (
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/models"
)

type Sandbox interface {
	GetBoxdir() string
	GetBoxId() int
	GetTimeLimit() int
	GetWallTimeLimit() int
	GetMemoryLimit() int
	GetFileSizeLimit() int
	GetMaxProcesses() int
	GetFilenamesInBox() []string

	MoveFileToBox(filepath string) error
	AddFile(filepath string) error
	ContainsFile(filepath string) bool
	GetFile(filename string) ([]byte, error)

	AddAllowedDirectory(dirpath string) error
	SetTimeLimitInMiliseconds(timeInMiliseconds int)
	SetWallTimeLimitInMiliseconds(timeInMiliseconds int)
	SetMemoryLimitInKilobytes(memoryInKilobytes int)

	BuildCommand(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) *command.CommandBuilder
	Execute(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) (models.SandboxExecutionResult, error)

	Cleanup() error
}
