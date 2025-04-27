package compiler

import (
	"testing"

	"github.com/Ceruvia/grader/internal/errorz"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestCreateNewCompiler(t *testing.T) {
	t.Run("it should build a c compiler", func(t *testing.T) {
		compiler, err := CreateNewCompiler("c", "none", "hello.c", "hello")

		want := &Compiler{
			Language: "c",
			Builder:  "none",
			Script:   "gcc hello.c -o hello",
		}

		utils.AssertNotError(t, err)
		utils.AssertDeep(t, compiler, want)
	})

	t.Run("it should return error when supplied an unsupported language", func(t *testing.T) {
		_, err := CreateNewCompiler("unsupported", "none", "none", "none")
		utils.AssertError(t, err, errorz.ErrLanguageUnsupported)
	})
}

func TestCompile(t *testing.T) {
	compiler, err := CreateNewCompiler("c", "none", "hello.c", "hello")
	utils.AssertNotError(t, err)

	err = compiler.Compile("compiler.go", "compiler_test.go")
	utils.AssertNotError(t, err)
}
