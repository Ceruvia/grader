package language_test

import (
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	language "github.com/Ceruvia/grader/internal/executor/languages"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestCreateNewExecutor(t *testing.T) {
	t.Run("it should make a new CExecutor if all file exists", func(t *testing.T) {
		fs := fstest.MapFS{
			"myworkdir/hello.c": &fstest.MapFile{Data: []byte("hello")},
			"myworkdir/1.in":    &fstest.MapFile{Data: []byte("world")},
			"myworkdir/2.in":    &fstest.MapFile{Data: []byte("world")},
			"myworkdir/1.out":   &fstest.MapFile{Data: []byte("world")},
			"myworkdir/2.out":   &fstest.MapFile{Data: []byte("world")},
		}

		exc, err := language.CreateNewCExecutor(
			fs,
			"myworkdir",
			[]string{"hello.c"},
			[]string{"1.in", "2.in"},
			[]string{"1.out", "2.out"},
		)

		utils.AssertNotError(t, err)
		utils.AssertDeep(t, exc, language.CExecutor{
			Workdir:     "myworkdir",
			BuildFiles:  []string{"hello.c"},
			InputFiles:  []string{"1.in", "2.in"},
			OutputFiles: []string{"1.out", "2.out"},
		})
	})

	t.Run("it should return error if workdir and files doesn't exist", func(t *testing.T) {
		fsys := fstest.MapFS{
			"myworkdir/1.in":  &fstest.MapFile{Data: []byte("world")},
			"myworkdir/2.out": &fstest.MapFile{Data: []byte("world")},
		}

		_, err := language.CreateNewCExecutor(
			fsys,
			"myworkdir",
			[]string{"hello.c"},
			[]string{"1.in", "2.in"},
			[]string{"1.out", "2.out"},
		)

		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("expected fs.ErrNotExist error, got %v", err)
		}
	})
}

func TestScriptArgs(t *testing.T) {
	t.Run("return script args without supplied flags", func(t *testing.T) {
		executor := language.CExecutor{
			Workdir:          "hello",
			BuildFiles:       []string{"hello.c"},
			InputFiles:       []string{},
			OutputFiles:      []string{},
			BinaryExecutable: "ex",
		}

		got := executor.ScriptArgs()
		want := []string{"hello.c", "-o", "ex"}

		utils.AssertDeep(t, got, want)
	})
}

func TestCompile(t *testing.T) {
	t.Run("it should compile a singular file", func(t *testing.T) {
		executor, err := language.CreateNewCExecutor(
			os.DirFS("."),
			"tests/c/hello",
			[]string{"hello.c"},
			[]string{},
			[]string{},
		)

		utils.AssertNotError(t, err)

		stdout, stderr, err := executor.Compile()

		t.Log(stdout)
		t.Log(stderr)

		utils.AssertNotError(t, err)
	})

	t.Run("it should compile multiple file", func(t *testing.T) {
		executor, err := language.CreateNewCExecutor(
			os.DirFS("."),
			"tests/c/multiple",
			[]string{"hello.c"},
			[]string{},
			[]string{},
		)

		utils.AssertNotError(t, err)

		stdout, stderr, err := executor.Compile()

		t.Log(stdout)
		t.Log(stderr)

		utils.AssertNotError(t, err)
	})

	t.Run("it should fail when compiling empty file", func(t *testing.T) {
		executor, err := language.CreateNewCExecutor(
			os.DirFS("."),
			"tests/c/empty",
			[]string{"empty.c"},
			[]string{},
			[]string{},
		)

		utils.AssertNotError(t, err)

		_, stderr, err := executor.Compile()

		if err == nil {
			t.Errorf("expected error, got none")
		}

		if !strings.Contains(stderr, "error: ld returned 1 exit status") {
			t.Errorf("expected error code of 1, got %+v", stderr)
		}
	})
}

// t.Run("it is able to compile to binary", func(t *testing.T) {})
// t.Run("it is able to run binary", func(t *testing.T) {})
// t.Run("it is able to run binary with input and give output", func(t *testing.T) {})
// t.Run("it is able to grade singular testcase", func(t *testing.T) {})
// t.Run("it is able to grade all testcase", func(t *testing.T) {})
