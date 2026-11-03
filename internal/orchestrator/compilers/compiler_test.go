package compilers_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func TestSourceFileCompiler(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		t.Run("Returns error when language provide is Nil", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{
				BoxId:  200,
				BoxDir: "/usr/local/bin/isolate/200/box",
			}

			_, err := compilers.PrepareSourceFileCompiler(&sbx, nil)
			tester.AssertCustomError(t, err, languages.ErrLanguageNotExist)
		})
	})
}
