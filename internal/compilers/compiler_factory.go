package compilers

import (
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func CreateCompilerBasedOnLang(sandbox sandboxes.Sandbox, language languages.Language) (Compiler, error) {
	languageName := language.GetName()
	var (
		comp Compiler
		err  error
	)

	switch languageName {
	case "c":
		comp, err = PrepareSingleSourceFileCompiler(sandbox, language)
	case "Java":
		comp, err = PrepareSingleSourceFileCompiler(sandbox, language)
	}

	return comp, err
}
