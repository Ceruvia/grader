package compiler

import (
	"errors"
	"os"

	"github.com/Ceruvia/grader/internal/errorz"
)

type Compiler struct {
	Language   string
	Builder    string
	Script     string
	InputPath  string
	OutputPath string
}

func CreateNewCompiler(language, builder, compileScript, runScript string) (*Compiler, error) {
	switch language {
	case "c":
		return &Compiler{
			Language: language,
			Builder:  builder,
			Script:   "gcc " + compileScript + " -o " + runScript,
		}, nil
	default:
		return nil, errorz.ErrLanguageUnsupported
	}
}

func (c *Compiler) Compile(inputPath, outputPath string) error {
	if _, err := os.Stat(inputPath); errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(outputPath); errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
