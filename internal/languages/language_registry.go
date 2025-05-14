package languages

import "github.com/Ceruvia/grader/internal/languages/clang"

var (
	CGradingLanguage = clang.CLanguage{}
	ELanguageNotExists = LanguageNotExists{}
)

func GetLanguageSimpleton(languageName string) (Language) {
	switch languageName {
	case "c":
		return CGradingLanguage
	default:
		return ELanguageNotExists
	}
}