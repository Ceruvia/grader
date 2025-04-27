package compiler

import (
	"errors"
	"os"
	"os/exec"

	"github.com/Ceruvia/grader/internal/errorz"
)

type Compiler struct {
	Language   string
	Compiler   string
	Builder    string
	InputFiles []string
	OutputName string
}

func CreateNewCompiler(language, builder string, inputFiles []string, outputName string) (*Compiler, error) {
	switch language {
	case "c":
		return &Compiler{
			Language:   language,
			Compiler:   "gcc",
			Builder:    builder,
			InputFiles: inputFiles,
			OutputName: outputName,
		}, nil
	default:
		return nil, errorz.ErrLanguageUnsupported
	}
}

func (c *Compiler) Compile(workDir string) error {
	if _, err := os.Stat(workDir); errors.Is(err, os.ErrNotExist) {
		return err
	}

	cmd := exec.Command(c.Compiler, c.ScriptArgs()...)
	cmd.Dir = workDir

	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
}

func (c *Compiler) ScriptArgs() []string {
	switch c.Language {
	case "c":
		return append(c.InputFiles, []string{"-o", c.OutputName}...)
	default:
		return []string{}
	}
}
