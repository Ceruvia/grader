package reader_test

import (
	"testing"
	"testing/fstest"

	"github.com/Ceruvia/grader/internal/reader"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestReadTestcaseInputOutputFile(t *testing.T) {
	fsys := fstest.MapFS{
		"firstdir/1.in":  &fstest.MapFile{Data: []byte("saya input pertama")},
		"firstdir/1.out": &fstest.MapFile{Data: []byte("saya output pertama")},
		"noinput/1.out":  &fstest.MapFile{Data: []byte("saya output pertama")},
		"nooutput/1.in":  &fstest.MapFile{Data: []byte("saya input pertama")},
		"emptydir":       &fstest.MapFile{},
	}

	t.Run("it should be able to read input and output", func(t *testing.T) {
		gotInput, gotOutput, err := reader.ReadTestcaseInputOutputFile(fsys, "firstdir", "1.in", "1.out")

		utils.AssertNotError(t, err)

		wantInput := "saya input pertama"
		wantOutput := "saya output pertama"

		if gotInput != wantInput {
			t.Errorf(`read input as %q when expectin %q`, gotInput, wantInput)
		}

		if gotOutput != wantOutput {
			t.Errorf(`read output as %q when expectin %q`, gotOutput, wantOutput)
		}
	})

	t.Run(`it should return error "no input file" found when input file isn't present`, func(t *testing.T) {
		_, _, err := reader.ReadTestcaseInputOutputFile(fsys, "noinput", "1.in", "1.out")

		utils.AssertError(t, err, reader.ErrInputNotExist)
	})

	t.Run(`it should return error "no ouput file" found when ouput file isn't present`, func(t *testing.T) {
		_, _, err := reader.ReadTestcaseInputOutputFile(fsys, "nooutput", "1.in", "1.out")

		utils.AssertError(t, err, reader.ErrOutputNotExist)
	})

	t.Run("it should error when no files are present", func(t *testing.T) {
		_, _, err := reader.ReadTestcaseInputOutputFile(fsys, "empty", "1.in", "1.out")

		if !(err == reader.ErrInputNotExist || err == reader.ErrOutputNotExist) {
			t.Errorf("got no error when expecting at least input or output error")
		}
	})
}

func BenchmarkReadTestcaseInputOutputFile(t *testing.B) {
	fsys := fstest.MapFS{
		"mydir/1.in":  &fstest.MapFile{Data: []byte("saya input pertama")},
		"mydir/1.out": &fstest.MapFile{Data: []byte("saya output pertama")},
	}

	t.ResetTimer()
	for t.Loop() {
		reader.ReadTestcaseInputOutputFile(fsys, "mydir", "1.in", "1.out")
	}
}
