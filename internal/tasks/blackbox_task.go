package tasks

import (
	"encoding/json"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
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
) (string, error) {
	acquiredSandbox := pool.Pool.Acquire()
	defer pool.Pool.Release(acquiredSandbox)

	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", acquiredSandbox.BoxId)
	if err != nil {
		return "", nil
	}

	if err := DownloadFileToSandbox(sbx, "grader.zip", graderFilesURL); err != nil {
		return "", nil
	}

	if err := DownloadFileToSandbox(sbx, "submission.zip", submissionFilesURL); err != nil {
		return "", nil
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

	b, err := json.Marshal(orchestrator.GradeBlackboxSubmission(sbx, submission))
	if err != nil {
		return "", err
	}

	return string(b), nil
}
