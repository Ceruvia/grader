package compiler

import (
	"errors"
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/errorz"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestCreateNewCompiler(t *testing.T) {
	t.Run("it should build a c compiler", func(t *testing.T) {
		compiler, err := CreateNewCompiler("c", "none", []string{"hello.c"}, "hello")

		want := &Compiler{
			Language:   "c",
			Compiler:   "gcc",
			Builder:    "none",
			InputFiles: []string{"hello.c"},
			OutputName: "hello",
		}

		utils.AssertNotError(t, err)
		utils.AssertDeep(t, compiler, want)
	})

	t.Run("it should return error when supplied an unsupported language", func(t *testing.T) {
		_, err := CreateNewCompiler("unsupported", "none", []string{}, "none")
		utils.AssertError(t, err, errorz.ErrLanguageUnsupported)
	})
}

func TestCompile(t *testing.T) {
	t.Run("compile c", func(t *testing.T) {
		compiler, err := CreateNewCompiler("c", "none", []string{"hello.c"}, "hello")
		utils.AssertNotError(t, err)

		err = compiler.Compile("test/c")
		utils.AssertNotError(t, err)

		if _, err := os.Stat("test/c/hello"); errors.Is(err, os.ErrNotExist) {
			t.Fatalf("compiled binary not found")
		}
	})
}

func TestScriptArgs(t *testing.T) {
	t.Run("c script", func(t *testing.T) {
		compiler, err := CreateNewCompiler("c", "none", []string{"hello.c", "hai.c"}, "hello")

		utils.AssertNotError(t, err)

		got := compiler.ScriptArgs()
		want := []string{"hello.c", "hai.c", "-o", "hello"}

		utils.AssertDeep(t, got, want)
	})
}
