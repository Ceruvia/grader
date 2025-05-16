package factory

import (
	"errors"

	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

type CreateCompilerFunction func(sandboxes.Sandbox, languages.Language) (compilers.Compiler, error)

var (
	GetFunction = map[string]CreateCompilerFunction{
		CGradingLanguage.GetName():    compilers.PrepareSingleSourceFileCompiler,
		JavaGradingLanguage.GetName(): compilers.PrepareSingleSourceFileCompiler,
	}
)

func CreateCompiler(sandbox sandboxes.Sandbox, language, builder string) (compilers.Compiler, error) {
	constructor := GetFunction[language]
	if constructor == nil {
		return nil, errors.New("Language or builder does not exist!")
	}

	if builder != "" {
		return constructor(sandbox, GetLanguage(builder))
	}
	return constructor(sandbox, GetLanguage(language))
}
