package compilers_test

import (
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSourceFileCompiler(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		t.Run("Returns SourceFileCompiler", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 200)
			if err != nil {
				t.Fatal(err)
			}
			defer sbx.Cleanup()

			clang := clang.CLanguage{}
			compiler, err := compilers.PrepareSourceFileCompiler(sbx, clang)
			tester.AssertNotError(t, err)

			tester.AssertDeep(t, compiler.GetSandbox().GetTimeLimit(), 20*1000)
			tester.AssertDeep(t, compiler.GetSandbox().GetMemoryLimit(), 1024*1024)
			tester.AssertDeep(t, compiler.GetRedirections().StandardErrorFilename, compilers.CompilationOutputFilename)
			tester.AssertDeep(t, compiler.GetRedirections().StandardOutputFilename, compilers.CompilationOutputFilename)
			tester.AssertDeep(t, compiler.GetRedirections().MetaFilename, "/var/local/lib/isolate/200/box/"+compilers.CompilationMetaFilename)
		})

		t.Run("Returns error when language provide is Nil", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{
				BoxId:  200,
				BoxDir: "/usr/local/bin/isolate/200/box",
			}

			_, err := compilers.PrepareSourceFileCompiler(&sbx, nil)
			tester.AssertCustomError(t, err, languages.ErrLanguageNotExist)
		})
	})

	t.Run("Compile", func(t *testing.T) {
		t.Run("Returns success when compiling C file", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 202)
			if err != nil {
				t.Fatal(err)
			}
			defer sbx.Cleanup()

			tempDir := "../../../tests/c_test/adt"
			sourceFiles := []string{"array.c", "array.h", "boolean.h", "ganjilgenap.c"}

			for _, file := range sourceFiles {
				err = sbx.AddFile(tempDir + "/" + file)
				tester.AssertNotError(t, err)
			}

			clang := clang.CLanguage{}

			compiler, err := compilers.PrepareSourceFileCompiler(sbx, clang)
			tester.AssertNotError(t, err)

			res := compiler.Compile("ganjilgenap.c", []string{"ganjilgenap.c", "array.c"})

			want := compilers.CompilerResult{
				IsSuccess:      true,
				BinaryFilename: "ganjilgenap",
				StdoutStderr:   "",
			}

			tester.AssertDeep(t, res, want)
		})

		t.Run("Returns failed when compiling uncompileable C file", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 203)
			if err != nil {
				t.Fatal(err)
			}
			// defer sbx.Cleanup()

			t.Log(sbx.BoxDir)

			tempDir := "../../../tests/c_test/uncompileable"
			sourceFiles := []string{"unfoundfunc.c"}

			for _, file := range sourceFiles {
				err = sbx.AddFile(tempDir + "/" + file)
				tester.AssertNotError(t, err)
			}

			clang := clang.CLanguage{}

			compiler, err := compilers.PrepareSourceFileCompiler(sbx, clang)
			tester.AssertNotError(t, err)

			res := compiler.Compile("unfoundfunc.c", []string{"unfoundfunc.c"})

			tester.AssertDeep(t, res.IsSuccess, false)
			if !strings.Contains(res.StdoutStderr, "error: implicit declaration of function ‘prinf’") {
				t.Errorf(`expected to get "implicit declaration of function" error in output, instead got %q`, res.StdoutStderr)
			}
		})
	})
}
