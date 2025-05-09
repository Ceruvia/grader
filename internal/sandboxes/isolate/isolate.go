package isolate

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/Ceruvia/grader/internal/command"
)

type IsolateSandbox struct {
	IsolatePath string
	BoxId       int
	AllowedDirs []string
	Filenames   []string

	BoxDir  string
	Command []string

	StandardInputFilename  string
	StandardOutputFilename string
	StandardErrorFilename  string
	MetaFilename           string

	TimeLimit     int
	WallTimeLimit int
	MemoryLimit   int
	FileSizeLimit int
	MaxProcesses  int
}

func CreateIsolateSandbox(isolatePath string, boxId int) (IsolateSandbox, error) {
	isolate := IsolateSandbox{
		IsolatePath:   isolatePath,
		BoxId:         boxId,
		AllowedDirs:   []string{},
		Filenames:     []string{},
		FileSizeLimit: 100 * 1024, // defaults to 100KB
		MaxProcesses:  50,         // defaults to 50 processes
	}

	err := isolate.initSandbox()
	if err != nil {
		return IsolateSandbox{}, err
	}

	return isolate, nil
}

func (s *IsolateSandbox) AddFile(filepath string) error {
	_, err := copy(filepath, s.BoxDir+"/"+parseFilenameFromPath(filepath))

	if err != nil {
		return err
	}

	s.Filenames = append(s.Filenames, parseFilenameFromPath(filepath))
	return nil
}

func (s *IsolateSandbox) ContainsFile(filename string) bool {
	return slices.Contains(s.Filenames, filename)
}

func (s *IsolateSandbox) GetFile(filename string) (fs.File, error) {
	// TODO: maybe custom error when file is not in s.Filenames
	// TODO: maybe balikin []byte aja ketimbang fs.File

	file, err := os.Open(appendBoxdir(s.BoxDir, filename))

	if err != nil {
		return nil, err
	}

	return file, nil
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

func (s *IsolateSandbox) BuildCommand(runCommand command.CommandBuilder) *command.CommandBuilder {
	return s.BuildCommandWithRedirections(runCommand, s.StandardInputFilename, s.StandardOutputFilename, s.StandardErrorFilename, s.MetaFilename)
}

// TODO; if useSavedRedirection then used the s.Redirections, else check if file exist or not
func (s *IsolateSandbox) BuildCommandWithRedirections(runCommand command.CommandBuilder, stdinFilepath, stdoutFilepath, stderrFilepath, metaFilepath string) *command.CommandBuilder {
	sandboxedCommand := command.GetCommandBuilder(s.IsolatePath)
	sandboxedCommand.AddArgs("-b " + strconv.Itoa(s.BoxId))

	for _, dir := range s.AllowedDirs {
		sandboxedCommand.AddArgs(fmt.Sprintf("--dir=%s:rw", dir))
	}

	sandboxedCommand.AddArgs("-e")

	if s.MaxProcesses > 0 {
		sandboxedCommand.AddArgs("--cg").AddArgs("--cg-timing").AddArgs(fmt.Sprintf("-p%d", s.MaxProcesses))
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

	if stdinFilepath != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-i%s", stdinFilepath))
	}

	if stdoutFilepath != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-o%s", stdoutFilepath))
	}

	if stderrFilepath != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-r%s", stderrFilepath))
	}

	if metaFilepath != "" {
		sandboxedCommand.AddArgs(fmt.Sprintf("-M%s", metaFilepath))
	}

	sandboxedCommand.AddArgs("--run").AddArgs("--")

	sandboxedCommand.AddArgs(runCommand.Program).AddArgs(runCommand.Args...)

	return sandboxedCommand
}

func (s *IsolateSandbox) Execute(command command.CommandBuilder) {
}

func (s *IsolateSandbox) ExecuteWithRedirections(command command.CommandBuilder, stdinFilepath, stdoutFilepath, stderrFilepath, metaFilepath string) {
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
		s.BoxDir = strings.Trim(string(stdout), "\n")
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

func copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
