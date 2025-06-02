package tasks

import (
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func GradeBlackbox(
	submissionId string,
	graderFilesURL, submissionFilesURL string,
	inputTestcases, outputTestcases []string,
	timeLimit, memoryLimit int,
	language string,

	mainSourceFilename string,
) (evaluator.GradingResult, error) {
	acquiredSandbox := pool.Pool.Acquire()
	defer pool.Pool.Release(acquiredSandbox)

	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", acquiredSandbox.BoxId)
	if err != nil {
		return evaluator.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, nil
	}

	if err := DownloadFileToSandbox(sbx, "grader.zip", graderFilesURL); err != nil {
		return evaluator.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, nil
	}

	if err := DownloadFileToSandbox(sbx, "submission.zip", submissionFilesURL); err != nil {
		return evaluator.GradingResult{
			Status:       "Internal Error",
			IsSuccess:    false,
			ErrorMessage: err.Error(),
		}, nil
	}

	submission := &models.SubmissionWithFiles{
		Core: models.Core{
			Id:       submissionId,
			Engine:   "blackbox",
			Language: language,
			Limits: models.GradingLimit{
				TimeInMiliseconds: timeLimit,
				MemoryInKilobytes: memoryLimit,
			},
			Testcases: createTestcases(inputTestcases, outputTestcases),
		},
		MainSourceFilename: mainSourceFilename,
	}

	return orchestrator.GradeBlackboxSubmission(sbx, submission), nil
}
