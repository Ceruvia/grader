package compiler

import (
	"errors"
	"log"
	"os"
	"os/exec"

	errorz "github.com/Ceruvia/grader/internal/errors"
)

type Compiler struct {
	Language   string
	Compiler   string
	Builder    string
	Files      []string
	OutputName string
}

func CreateNewCompiler(language, builder string, files []string, outputName string) (*Compiler, error) {
	switch language {
	case "c":
		return &Compiler{
			Language:   language,
			Compiler:   "gcc",
			Builder:    builder,
			Files:      files,
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
	
	log.Println(c.ScriptArgs())
	return nil
}

func (c *Compiler) ScriptArgs() []string {
	switch c.Language {
	case "c":
		return append(c.Files, []string{"-o", c.OutputName}...)
	default:
		return []string{}
	}
}
