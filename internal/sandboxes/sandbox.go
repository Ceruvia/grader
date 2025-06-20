package sandboxes

import (
	"errors"

	"github.com/Ceruvia/grader/internal/helper/command"
)

var (
	ErrFilenameNotInBox = errors.New("Filename not found in box!")
	ErrSandboxIsNil = errors.New("Sandbox is a null pointer!")
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

	AddFile(filepath string) error
	AddFileWithoutMove(filepath string)
	MoveFileToBox(filepath string) error
	ContainsFile(filepath string) bool
	GetFile(filename string) ([]byte, error)

	AddAllowedDirectory(dirpath string) error
	SetTimeLimitInMiliseconds(timeInMiliseconds int)
	SetWallTimeLimitInMiliseconds(timeInMiliseconds int)
	SetMemoryLimitInKilobytes(memoryInKilobytes int)

	BuildCommand(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) *command.CommandBuilder
	Execute(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) SandboxExecutionResult

	Cleanup() error
}
