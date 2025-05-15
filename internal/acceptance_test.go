package internal_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal"
	"github.com/Ceruvia/grader/internal/models"
)

func TestGradingC(t *testing.T) {
	t.Run("it should be able to compile, run, and grade a simple Hello, World code", func(t *testing.T) {
		submission := models.Submission{
			Id:            "awjofi92",
			TempDir:       "../tests/c_test/hello",
			Language:      "c",
			TCInputFiles:  []string{"1.in"},
			TCOutputFiles: []string{"1.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		internal.GradeSubmission(990, submission)
	})

	t.Run("it should be able to compile, run, and grade an ADT question", func(t *testing.T) {
		submission := models.Submission{
			Id:            "dhsai82d",
			TempDir:       "../tests/c_test/adt",
			Language:      "c",
			TCInputFiles:  []string{"1.in", "2.in", "3.in", "4.in", "5.in", "6.in", "7.in", "8.in", "9.in", "10.in"},
			TCOutputFiles: []string{"1.out", "2.out", "3.out", "4.out", "5.out", "6.out", "7.out", "8.out", "9.in", "10.out"},
			Limits: models.GradingLimit{
				TimeInMiliseconds: 1000,
				MemoryInKilobytes: 102400,
			},
		}

		internal.GradeSubmission(990, submission)
	})
}
