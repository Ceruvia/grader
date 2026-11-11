package sandboxes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Ceruvia/grader/internal/helper"
	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/helper/files"
)

// runs bounds every box run of this process.
var runs = context.Background()

// BoundRuns makes the end of ctx kill every running box and fail every later
// run. A grader binds it to the release of its jobs at start, before any run:
// a stop that releases the jobs ends their runs with the same cancel, so a run
// it killed is always seen as released and never becomes a verdict.
func BoundRuns(ctx context.Context) { runs = ctx }

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
		FileSizeLimit: 100 * 1024, // 100 MiB per file, with a 256 MiB aggregate tmpfs bound
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
func (s *IsolateSandbox) GetFilenamesInBox() []string { return helper.RemoveDuplicates(s.Filenames) }

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

	return ReadBoxFile(s, filename, 0)
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

	for _, dir := range s.AllowedDirs {
		sandboxedCommand.AddArgs(fmt.Sprintf("--dir=%s:rw", dir))
	}
	// /tmp and /box are two views of the SAME bounded filesystem. A separate
	// scratch mount would let a program bypass the aggregate byte/inode quota.
	sandboxedCommand.AddArgs("--dir=/tmp=./box:rw")
	// Keep ordinary devices, but never inherit writable /dev/shm, mqueue or
	// devpts submounts from the privileged grader container.
	sandboxedCommand.AddArgs("--dir=/dev:norec:dev")

	sandboxedCommand.AddArgs("-E", "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	sandboxedCommand.AddArgs("-E", "HOME=/box")
	sandboxedCommand.AddArgs("-E", "LANG=C.UTF-8")

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

func (s *IsolateSandbox) Execute(runCommand command.CommandBuilder, redirectionFiles RedirectionFiles) SandboxExecutionResult {
	command := s.BuildCommand(runCommand, redirectionFiles)

	// No shell: the program is the operator's isolate binary, and every argument that can carry a
	// submission filename follows `--run --`, so isolate runs it inside the box and never parses it.
	cmd := exec.CommandContext(runs, command.Program, command.Args...) //nolint:gosec // see above.
	// SIGTERM, not Go's default SIGKILL: isolate then kills the box and writes its meta file itself. The
	// keeper's child, the proxy, is init of the box PID namespace, so the program dies when the proxy does.
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	// A keeper that ignores the SIGTERM gets SIGKILL, so a released run never hangs.
	cmd.WaitDelay = 2 * time.Second
	var output diagnostics.Buffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) || exitError.ExitCode() != 1 {
			return SandboxExecutionResult{
				Status: INTERNAL_ERROR, ExitCode: -1, Time: -1, Memory: -1,
				Message: diagnostics.Limit(fmt.Sprintf("isolate launch/run failed: %v: %s", err, output.String())),
			}
		}
		// Isolate's exit 1 reports a student-program failure in the metadata.
	}

	res, err := ParseMetaResult(redirectionFiles.MetaFilename)
	if err != nil {
		return SandboxExecutionResult{
			Status:  PARSING_META_ERROR,
			Time:    -1,
			Memory:  -1,
			Message: err.Error(),
		}
	}
	if cmd.ProcessState.ExitCode() != 0 && res.Status == ZERO_EXIT_CODE {
		return SandboxExecutionResult{
			Status: INTERNAL_ERROR, ExitCode: -1, Time: -1, Memory: -1,
			Message: "Isolate failed but metadata reported success",
		}
	}

	return res
}

func (s *IsolateSandbox) Cleanup() error {
	if err := unmountBoxStorage(s.BoxDir); err != nil {
		return err
	}
	err := s.cleanUpIsolate()
	if err != nil {
		return err
	}
	return os.RemoveAll(TaskDir(s.BoxId))
}

// CleanBox removes whatever a previous grader process left in the box, live
// processes included. It does nothing to a box that was never initialized.
func CleanBox(isolatePath string, boxId int) error {
	if err := KillBoxProcesses(boxId); err != nil {
		return fmt.Errorf("kill box processes: %w", err)
	}
	if err := recoverBoxStorage(boxId); err != nil {
		return err
	}
	stale := IsolateSandbox{IsolatePath: isolatePath, BoxId: boxId}
	// The keeper releases the box lock a moment after its program is killed.
	var err error
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if err = stale.cleanUpIsolate(); err == nil {
			return os.RemoveAll(TaskDir(boxId))
		}
		// --cleanup rejects a box whose --init failed half way. --init recovers it.
		if _, initErr := stale.initIsolate(); initErr != nil {
			err = errors.Join(err, initErr)
		}
	}
	return err
}

func (s *IsolateSandbox) initSandbox() error {
	if err := recoverBoxStorage(s.BoxId); err != nil {
		return err
	}
	root, err := s.initIsolate()
	if err != nil {
		return err
	}
	s.BoxDir = root + "/box"
	if err := mountBoxStorage(s.BoxDir); err != nil {
		return errors.Join(fmt.Errorf("mount bounded box storage: %w", err), s.cleanUpIsolate())
	}
	return nil
}

// initIsolate runs isolate --init and returns the box root it prints.
func (s *IsolateSandbox) initIsolate() (string, error) {
	command := command.GetCommandBuilder(s.IsolatePath)
	command.AddArgs("-b " + strconv.Itoa(s.BoxId))
	command.AddArgs("--cg")
	command.AddArgs("--init")

	cmd := exec.Command(command.Program, command.Args...) //nolint:gosec // operator's isolate binary, args are the box id and fixed flags.
	var stdout diagnostics.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("initialize isolate: %w: %s", err, stdout.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func (s *IsolateSandbox) cleanUpIsolate() error {
	commandArgs := []string{"-b", strconv.Itoa(s.BoxId), "--cg", "--cleanup"}

	cmd := exec.Command(s.IsolatePath, commandArgs...) //nolint:gosec // operator's isolate binary, args are the box id and fixed flags.
	var stdout diagnostics.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clean up isolate: %w: %s", err, stdout.String())
	}
	return nil
}

func parseFilenameFromPath(filepath string) string {
	splitted := strings.Split(filepath, "/")
	return splitted[len(splitted)-1]
}

func appendBoxdir(boxdir, filename string) string {
	return boxdir + "/" + filename
}
