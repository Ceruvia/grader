package languages

import (
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/languages/javalang"
)

var (
	CGradingLanguage    = clang.CLanguage{}
	JavaGradingLanguage = javalang.JavaLanguage{}
	ELanguageNotExists  = LanguageNotExists{}
)

func GetLanguageSimpleton(languageName string) Language {
	switch languageName {
	case "c":
		return CGradingLanguage
	case "java":
		return JavaGradingLanguage
	default:
		return ELanguageNotExists
	}
}
