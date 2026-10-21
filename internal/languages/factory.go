package languages

import (
	"github.com/Ceruvia/grader/internal/languages/builder"
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/languages/cpplang"
	"github.com/Ceruvia/grader/internal/languages/haskelllang"
	"github.com/Ceruvia/grader/internal/languages/javalang"
	"github.com/Ceruvia/grader/internal/languages/pylang"
)

var (
	CGradingLanguage       = clang.CLanguage{}
	HaskellGradingLanguage = haskelllang.HaskellLanguage{}
	Cpp11GradingLanguage   = cpplang.Cpp11Language{}
	Cpp17GradingLanguage   = cpplang.Cpp17Language{}
	Cpp20GradingLanguage   = cpplang.Cpp20Language{}
	JavaGradingLanguage    = javalang.JavaLanguage{}
	Python3GradingLanguage = pylang.Python3Language{}
	MakefileBuilder        = builder.MakefileBuilder{}

	LanguageSimpleton = map[string]Language{
		CGradingLanguage.GetName():       CGradingLanguage,
		HaskellGradingLanguage.GetName(): HaskellGradingLanguage,
		Cpp11GradingLanguage.GetName():   Cpp11GradingLanguage,
		Cpp17GradingLanguage.GetName():   Cpp17GradingLanguage,
		Cpp20GradingLanguage.GetName():   Cpp20GradingLanguage,
		JavaGradingLanguage.GetName():    JavaGradingLanguage,
		Python3GradingLanguage.GetName(): Python3GradingLanguage,
		MakefileBuilder.GetName():        MakefileBuilder,
	}
)

func GetLanguage(languageSimpleName string) Language {
	language := LanguageSimpleton[languageSimpleName]
	return language
}
