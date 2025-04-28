package orchestrator

import (
	"github.com/Ceruvia/grader/internal/compiler"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/runner"
)

func Execute(submission models.Submission) float32 {
	compiler, err := compiler.CreateNewCompiler(
		submission.Language,
		submission.Builder,
		submission.Files,
		"a")

	if err != nil {
		panic(err)
	}

	compiler.Compile(submission.Workdir)

	for _, tc := range submission.Testcases {
		runner.RunWithInputAndOutput(submission.Workdir+"/a",
			submission.Workdir+"/"+tc.InputFilename,
			submission.Workdir+"/"+tc.ExpectedOutputFilename+".actual")
	}

	total := 0
	for _, tc := range submission.Testcases {
		res, _ := runner.Grade(
			submission.Workdir+"/"+tc.ExpectedOutputFilename,
			submission.Workdir+"/"+tc.ExpectedOutputFilename+".actual",
		)
		if res {
			total++
		}
	}

	return float32(total) / float32(len(submission.Testcases))
}
