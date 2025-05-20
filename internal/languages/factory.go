package languages

import (
	"github.com/Ceruvia/grader/internal/languages/builder"
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/languages/javalang"
	"github.com/Ceruvia/grader/internal/languages/pylang"
)

var (
	CGradingLanguage       = clang.CLanguage{}
	JavaGradingLanguage    = javalang.JavaLanguage{}
	Python3GradingLanguage = pylang.Python3Language{}
	MakefileBuilder        = builder.MakefileBuilder{}

	LanguageSimpleton = map[string]Language{
		CGradingLanguage.GetName():       CGradingLanguage,
		JavaGradingLanguage.GetName():    JavaGradingLanguage,
		Python3GradingLanguage.GetName(): Python3GradingLanguage,
		MakefileBuilder.GetName():        MakefileBuilder,
	}
)

func GetLanguage(languageSimpleName string) Language {
	language := LanguageSimpleton[languageSimpleName]
	return language
}
