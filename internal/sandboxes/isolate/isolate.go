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
)

type IsolateSandbox struct {
	IsolatePath string
	BoxId       int
	AllowedDirs []string
	Filenames   []string

	BoxDir  string
	Command []string

	StandardInput  []byte
	StandardOutput []byte
	StandardError  []byte

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



func (s *IsolateSandbox) Cleanup() error {
	err := s.cleanUpIsolate()
	return err
}

func (s *IsolateSandbox) initSandbox() error {
	commandArgs := []string{"-b", strconv.Itoa(s.BoxId), "--cg", "--init"}

	cmd := exec.Command(s.IsolatePath, commandArgs...)
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
