package orchestrator

import (
	"testing"

	"github.com/Ceruvia/grader/internal/models"
)

func TestExecute(t *testing.T) {
	submission := models.Submission{
		Language:   "c",
		UseBuilder: false,
		Files:      []string{"test.c"},
		Testcases: []models.Testcase{
			{InputFilename: "1.in", ExpectedOutputFilename: "1.out"},
			{InputFilename: "2.in", ExpectedOutputFilename: "2.out"},
		},

		Workdir: "test/c_1",
	}

	res := Execute(submission)
	if res != 2 {
		t.Errorf("expected 2 tc right, got %+v", res)
	}
}
