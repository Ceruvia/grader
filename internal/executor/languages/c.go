package language

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"os/exec"
	"sort"
	"sync"
	"time"

	"github.com/Ceruvia/grader/internal/models"
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

func (exc *CExecutor) Run(stdin io.Reader, stdout, stderr io.Writer) error {
	if exc.BinaryExecutable == "" {
		return errors.New("dumbass")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "./"+exc.BinaryExecutable)
	cmd.Dir = exc.Workdir
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return ctx.Err()
	}

	return err
}

func (exc *CExecutor) RunAgainstTestcase(input, expectedOutput string) (models.Verdict, string, string, error) {
	var stdinBuf, stdoutBuf, stderrBuf bytes.Buffer
	stdinBuf.Write([]byte(input))

	if err := exc.Run(&stdinBuf, &stdoutBuf, &stderrBuf); err != nil {
		if err == context.DeadlineExceeded {
			return models.VerdictTLE, stdoutBuf.String(), stderrBuf.String(), err
		}
		return models.VerdictRE, stdoutBuf.String(), stderrBuf.String(), err
	}

	verdict := models.VerdictWA
	if stdoutBuf.String() == expectedOutput {
		verdict = models.VerdictAC
	}

	return verdict, stdoutBuf.String(), stderrBuf.String(), nil
}

func (exc *CExecutor) GradeAll() []models.Verdict {
	type GradingVerdict struct {
		models.Verdict
		int
	}

	result := make(chan GradingVerdict)

	lenOfTC := len(exc.InputFiles)

	for i := range lenOfTC {
		go func() {
			// TODO: i should really handle all these errors :D

			input, expectedOutput, _ := exc.ReadInputOutputFile(i)
			verdict, _, _, _ := exc.RunAgainstTestcase(input, expectedOutput)
			result <- GradingVerdict{verdict, i}
		}()
	}

	verdicts := []GradingVerdict{}
	for range lenOfTC {
		v := <-result
		verdicts = append(verdicts, v)
	}

	sort.Slice(verdicts, func(i, j int) bool {
		return verdicts[i].int < verdicts[j].int
	})

	verdictOnly := []models.Verdict{}
	for _, v := range verdicts {
		verdictOnly = append(verdictOnly, v.Verdict)
	}

	return verdictOnly
}

func (exc *CExecutor) ScriptArgs() []string {
	return append(exc.BuildFiles, []string{"-o", exc.BinaryExecutable}...)
}

func (exc *CExecutor) AddWorkdirPrefix(s string) string {
	return exc.Workdir + "/" + s
}

func (exc *CExecutor) ReadInputOutputFile(idx int) (string, string, error) {
	var wg sync.WaitGroup

	contents := make([]string, 2)
	errors := make([]error, 2)

	for i, path := range []string{exc.AddWorkdirPrefix(exc.InputFiles[idx]), exc.AddWorkdirPrefix(exc.OutputFiles[idx])} {
		wg.Add(1)

		go func(idx int, p string) {
			defer wg.Done()

			data, err := ioutil.ReadFile(p)
			if err != nil {
				errors[idx] = err
				return
			}

			contents[idx] = string(data)
		}(i, path)
	}

	wg.Wait()

	inputData := contents[0]
	inputError := errors[0]
	outputData := contents[1]
	outputError := errors[1]

	if inputError != nil {
		return "", "", inputError
	}

	if outputError != nil {
		return "", "", inputError
	}

	return inputData, outputData, nil
}
