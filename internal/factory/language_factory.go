package factory

import (
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/languages/builder"
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/languages/javalang"
)

var (
	CGradingLanguage    = clang.CLanguage{}
	JavaGradingLanguage = javalang.JavaLanguage{}
	MakefileBuilder     = builder.MakefileBuilder{}

	LanguageSimpleton = map[string]languages.Language{
		CGradingLanguage.GetName():    CGradingLanguage,
		JavaGradingLanguage.GetName(): JavaGradingLanguage,
		MakefileBuilder.GetName():     MakefileBuilder,
	}
)

func GetLanguage(languageSimpleName string) languages.Language {
	language := LanguageSimpleton[languageSimpleName]
	return language
}
