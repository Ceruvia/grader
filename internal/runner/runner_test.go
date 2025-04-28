package runner

import (
	"errors"
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/utils"
)

func TestRun(t *testing.T) {
	t.Run("run c binary", func(t *testing.T) {
		got, err := Run("test/c/hello")
		utils.AssertNotError(t, err)

		want := "Hello, world!\n"

		utils.AssertDeep(t, got, want)
	})

	t.Run("run c binary with input", func(t *testing.T) {
		ok, err := RunWithInputAndOutput("test/c/input", "test/c/input.in", "test/c/actual.out")
		if !ok {
			t.Fatal(err)
		}

		if _, err := os.Stat("test/c/actual.out"); errors.Is(err, os.ErrNotExist) {
			t.Fatalf("actual output not created")
		}
	})
}

func TestGrade(t *testing.T) {
	t.Run("grade singular testcase", func(t *testing.T) {
		correct, err := Grade("test/c/expected.out", "test/c/actual.out")
		if err != nil {
			t.Fatal(err)
		}

		if !correct {
			t.Error("output should be correct")
		}
	})
}
