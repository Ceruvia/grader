package clang_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages/clang"
)

var (
	CGradingLanguage = clang.CLanguage{}
)

func TestGetName(t *testing.T) {
	got := CGradingLanguage.GetName()
	want := "C"
	tester.AssertDeep(t, got, want)
}

func TestGetAllowedExtention(t *testing.T) {
	got := CGradingLanguage.GetAllowedExtention()
	want := []string{"c"}
	tester.AssertDeep(t, got, want)
}

func TestGetCompilationCommand(t *testing.T) {
	Tests := []struct {
		mainSourceFilename string
		sourceFilenames    []string
		want               string
	}{
		{"main.c", []string{"main.c"}, "/usr/bin/gcc -std=gnu99 -o main main.c -O2 -lm"},
		{"grader.c", []string{"grader.c", "array.c", "boolean.c"}, "/usr/bin/gcc -std=gnu99 -o grader grader.c array.c boolean.c -O2 -lm"},
	}

	for _, test := range Tests {
		t.Run("it should be able to give the compilation command", func(t *testing.T) {
			got := CGradingLanguage.GetCompilationCommand(test.mainSourceFilename, test.sourceFilenames...)
			tester.AssertDeep(t, got.BuildFullCommand(), test.want)
		})
	}
}

func TestGetExecutionCommand(t *testing.T) {
	got := CGradingLanguage.GetExecutionCommand("main.c")
	want := "./main "
	tester.AssertDeep(t, got.BuildFullCommand(), want)
}

func TestGetExecutableFilename(t *testing.T) {
	got := CGradingLanguage.GetExecutableFilename("main.c")
	want := "main"
	tester.AssertDeep(t, got, want)
}
