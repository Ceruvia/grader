package sandboxes

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
	"github.com/Ceruvia/grader/internal/models"
)

type IsolateSandbox struct {
	IsolatePath string
	BoxId       int
	AllowedDirs []string
	Filenames   []string

	BoxDir string

	TimeLimit     int
	WallTimeLimit int
	MemoryLimit   int
	FileSizeLimit int
	MaxProcesses  int
}

func CreateIsolateSandbox(isolatePath string, boxId int) (*IsolateSandbox, error) {
	isolate := IsolateSandbox{
		IsolatePath:   isolatePath,
		BoxId:         boxId,
		AllowedDirs:   []string{},
		Filenames:     []string{},
		FileSizeLimit: 100 * 1024, // defaults to 100KB
		MaxProcesses:  50,         // defaults to 50 processes
	}

	if err := isolate.initSandbox(); err != nil {
		return nil, err
	}

	return &isolate, nil
}

func (s *IsolateSandbox) GetBoxdir() string           { return s.BoxDir }
func (s *IsolateSandbox) GetBoxId() int               { return s.BoxId }
func (s *IsolateSandbox) GetTimeLimit() int           { return s.TimeLimit }
func (s *IsolateSandbox) GetWallTimeLimit() int       { return s.WallTimeLimit }
func (s *IsolateSandbox) GetMemoryLimit() int         { return s.MemoryLimit }
func (s *IsolateSandbox) GetFileSizeLimit() int       { return s.FileSizeLimit }
func (s *IsolateSandbox) GetMaxProcesses() int        { return s.MaxProcesses }
func (s *IsolateSandbox) GetFilenamesInBox() []string { return s.Filenames }

func (s *IsolateSandbox) AddFile(filepath string) error {
	if err := s.MoveFileToBox(filepath); err != nil {
		return err
	}

	s.addFileToFilenames(filepath)
	return nil
}

func (s *IsolateSandbox) AddFileWithoutMove(filepath string) {
	s.addFileToFilenames(filepath)
}

func (s *IsolateSandbox) addFileToFilenames(filepath string) {
	s.Filenames = append(s.Filenames, parseFilenameFromPath(filepath))
}

func (s *IsolateSandbox) MoveFileToBox(filepath string) error {
	_, err := files.CopyFile(filepath, s.BoxDir+"/"+parseFilenameFromPath(filepath))
	return err
}

func (s *IsolateSandbox) ContainsFile(filename string) bool {
	return slices.Contains(s.Filenames, filename)
}

func (s *IsolateSandbox) GetFile(filename string) ([]byte, error) {
	if !s.ContainsFile(filename) {
		return nil, ErrFilenameNotInBox
	}

	data, err := os.ReadFile(appendBoxdir(s.BoxDir, filename))
	return data, err
}

func (s *IsolateSandbox) AddAllowedDirectory(dirpath string) error {
	if _, err := os.Stat(dirpath); err != nil {
		return err
	}

	s.AllowedDirs = append(s.AllowedDirs, dirpath)

	return nil
}

func (s *IsolateSandbox) SetTimeLimitInMiliseconds(timeInMiliseconds int) {
	s.TimeLimit = timeInMiliseconds
}

func (s *IsolateSandbox) SetWallTimeLimitInMiliseconds(timeInMiliseconds int) {
	s.WallTimeLimit = timeInMiliseconds
}

func (s *IsolateSandbox) SetMemoryLimitInKilobytes(memoryInKilobytes int) {
	s.MemoryLimit = memoryInKilobytes
}

func (s *IsolateSandbox) BuildCommand(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) *command.CommandBuilder {
	sandboxedCommand := command.GetCommandBuilder(s.IsolatePath)
	sandboxedCommand.AddArgs("-b " + strconv.Itoa(s.BoxId))

	sandboxedCommand.AddArgs("--dir=/etc")
	for _, dir := range s.AllowedDirs {
		sandboxedCommand.AddArgs(fmt.Sprintf("--dir=%s:rw", dir))
	}

	sandboxedCommand.AddArgs("-e")

	if s.MaxProcesses > 0 {
		sandboxedCommand.AddArgs("--cg").AddArgs(fmt.Sprintf("-p%d", s.MaxProcesses))
	}

	if s.TimeLimit > 0 {
		timeLimitInSeconds := float64(s.TimeLimit) / 1000
		sandboxedCommand.AddArgs(fmt.Sprintf("-t%g", timeLimitInSeconds))
		sandboxedCommand.AddArgs("-x0.5")
	}

	if s.WallTimeLimit > 0 {
		timeLimitInSeconds := float64(s.WallTimeLimit) / 1000
		sandboxedCommand.AddArgs(fmt.Sprintf("-w%g", timeLimitInSeconds))
	}

	if s.MemoryLimit > 0 {
		if s.MaxProcesses > 1 {
			sandboxedCommand.AddArgs(fmt.Sprintf("--cg-mem=%d", s.MemoryLimit))
		} else {
			sandboxedCommand.AddArgs(fmt.Sprintf("-m%d", s.MemoryLimit))
		}
		sandboxedCommand.AddArgs(fmt.Sprintf("-k%d", s.MemoryLimit))
	}

	if s.FileSizeLimit > 0 {
		sandboxedCommand.AddArgs(fmt.Sprintf("-f%d", s.FileSizeLimit))
	}

	if redirectionFiles.StandardInputFilename != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-i%s", redirectionFiles.StandardInputFilename))
	}

	if redirectionFiles.StandardOutputFilename != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-o%s", redirectionFiles.StandardOutputFilename))
	}

	if redirectionFiles.StandardErrorFilename != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-r%s", redirectionFiles.StandardErrorFilename))
	}

	if redirectionFiles.MetaFilename != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-M%s", redirectionFiles.MetaFilename))
	}

	sandboxedCommand.AddArgs("--run").AddArgs("--")

	sandboxedCommand.AddArgs(runCommand.Program).AddArgs(runCommand.Args...)

	return sandboxedCommand
}

func (s *IsolateSandbox) Execute(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) (models.SandboxExecutionResult, error) {
	command := s.BuildCommand(runCommand, redirectionFiles)

	cmd := exec.Command(command.Program, command.Args...)
	_, err := cmd.CombinedOutput()

	exitError, ok := err.(*exec.ExitError)

	if exitError != nil && !ok {
		return models.SandboxExecutionResult{
			Status:   models.INTERNAL_ERROR,
			ExitCode: exitError.ExitCode(),
			Time:     -1,
			Memory:   -1,
			Message:  exitError.Error(),
		}, exitError
	}

	res, err := ParseMetaResult(redirectionFiles.MetaFilename)
	if err != nil {
		return models.SandboxExecutionResult{
			Status:  models.PARSING_META_ERROR,
			Time:    -1,
			Memory:  -1,
			Message: err.Error(),
		}, err
	}

	return res, nil
}

func (s *IsolateSandbox) Cleanup() error {
	err := s.cleanUpIsolate()
	return err
}

func (s *IsolateSandbox) initSandbox() error {
	command := command.GetCommandBuilder(s.IsolatePath)
	command.AddArgs("-b " + strconv.Itoa(s.BoxId))
	command.AddArgs("--cg")
	command.AddArgs("--init")

	cmd := exec.Command(command.Program, command.Args...)
	stdout, err := cmd.CombinedOutput()

	if err == nil && cmd.ProcessState.ExitCode() == 0 {
		s.BoxDir = strings.Trim(string(stdout), "\n") + "/box"
		return nil
	}

	if strings.Contains(err.Error(), "Box already exists") {
		s.cleanUpIsolate()
	}

	if string(stdout) != "" {
		err = errors.New(string(stdout))
	}

	return err
}

func (s *IsolateSandbox) cleanUpIsolate() error {
	commandArgs := []string{"-b", strconv.Itoa(s.BoxId), "--cg", "--cleanup"}

	cmd := exec.Command(s.IsolatePath, commandArgs...)
	stdout, err := cmd.CombinedOutput()

	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		return errors.New("Cannot clean up Isolate! " + string(stdout))
	} else {
		return err
	}
}

func parseFilenameFromPath(filepath string) string {
	splitted := strings.Split(filepath, "/")
	return splitted[len(splitted)-1]
}

func appendBoxdir(boxdir, filename string) string {
	return boxdir + "/" + filename
}
