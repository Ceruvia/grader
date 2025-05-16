package compilers_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/compilers"
	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages/clang"
	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
)

func TestNormalCompiler(t *testing.T) {
	t.Run("it should be able to initialize a Normal compiler", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 13)
		if err != nil {
			t.Fatal(err)
		}
		defer sbx.Cleanup()

		clang := clang.CLanguage{}

		compiler, err := compilers.PrepareSingleSourceFileCompiler(&sbx, clang)
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}

		tester.AssertDeep(t, compiler.GetSandbox().GetTimeLimit(), 20*1000)
		tester.AssertDeep(t, compiler.GetSandbox().GetMemoryLimit(), 1024*1024)
		tester.AssertDeep(t, compiler.GetRedirections().StandardErrorFilename, compilers.CompilationOutputFilename)
		tester.AssertDeep(t, compiler.GetRedirections().StandardOutputFilename, compilers.CompilationOutputFilename)
		tester.AssertDeep(t, compiler.GetRedirections().MetaFilename, "/var/local/lib/isolate/13/box/"+compilers.CompilationMetaFilename)
	})

	t.Run("it should be able to compile a c language", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 14)
		if err != nil {
			t.Fatal(err)
		}
		defer sbx.Cleanup()

		// add source files to boxdir
		err = sbx.AddFile("../../tests/c_test/adt/array.c")
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}
		err = sbx.AddFile("../../tests/c_test/adt/array.h")
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}
		err = sbx.AddFile("../../tests/c_test/adt/boolean.h")
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}
		err = sbx.AddFile("../../tests/c_test/adt/ganjilgenap.c")
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}

		clang := clang.CLanguage{}

		compiler, err := compilers.PrepareSingleSourceFileCompiler(&sbx, clang)
		if err != nil {
			t.Errorf("expected got no error, but got %q", err)
		}

		res, err := compiler.Compile("ganjilgenap.c", []string{"ganjilgenap.c", "array.c"})

		want := compilers.CompilerResult{
			IsSuccess:      true,
			BinaryFilename: "ganjilgenap",
			StdoutStderr:   "",
		}

		tester.AssertDeep(t, res, want)
	})
}
