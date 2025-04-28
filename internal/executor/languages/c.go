package language

import (
	"bytes"
	"context"
	"io/fs"
	"os/exec"
	"time"
)

type CExecutor struct {
	Workdir     string
	BuildFiles  []string
	InputFiles  []string
	OutputFiles []string

	BinaryExecutable string
}

func CreateNewCExecutor(fsys fs.FS, workdir string, buildFiles []string, inputFiles []string, outputFiles []string) (CExecutor, error) {
	// check workdir exists
	if _, err := fs.Stat(fsys, workdir); err != nil {
		return CExecutor{}, err
	}
	// check all build files exist
	for _, buildFile := range buildFiles {
		if _, err := fs.Stat(fsys, workdir+"/"+buildFile); err != nil {
			return CExecutor{}, err
		}
	}
	// check all input files exist
	for _, inputFile := range inputFiles {
		if _, err := fs.Stat(fsys, workdir+"/"+inputFile); err != nil {
			return CExecutor{}, err
		}
	}
	// check all output files exist
	for _, outputFile := range outputFiles {
		if _, err := fs.Stat(fsys, workdir+"/"+outputFile); err != nil {
			return CExecutor{}, err
		}
	}

	return CExecutor{
		Workdir:     workdir,
		BuildFiles:  buildFiles,
		InputFiles:  inputFiles,
		OutputFiles: outputFiles,
	}, nil
}

func (exc *CExecutor) Execute() error {
	return nil
}

func (exc *CExecutor) Compile() (string, string, error) {
	// TODO: Redundant name, might be changed to user supplied name
	exc.BinaryExecutable = "test_ex" 

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "gcc", exc.ScriptArgs()...)
	cmd.Dir = exc.Workdir

	// attach stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return stdoutBuf.String(), stderrBuf.String(), ctx.Err()
	}

	return stdoutBuf.String(), stderrBuf.String(), err
}

func (exc *CExecutor) Run() error {
	return nil
}

func (exc *CExecutor) RunWithInput() error {
	return nil
}

func (exc *CExecutor) Grade() error {
	return nil
}

func (exc *CExecutor) GradeAll() error {
	return nil
}

func (exc *CExecutor) ScriptArgs() []string {
	return append(exc.BuildFiles, []string{"-o", exc.BinaryExecutable}...)
}
