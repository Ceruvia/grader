package tasks

import (
	"encoding/json"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
	log "github.com/sirupsen/logrus"
)

func GradeBlackbox(
	submissionId string,
	graderFilesURL, submissionFilesURL string,
	inputTestcases, outputTestcases []string,
	timeLimit, memoryLimit int,
	language string,
	mainSourceFilename string,
) (string, error) {
	log.WithFields(log.Fields{
		"submissionId":  submissionId,
		"language":      language,
		"graderURL":     graderFilesURL,
		"submissionURL": submissionFilesURL,
		"testcases":     len(inputTestcases),
		"mainFile":      mainSourceFilename,
	}).Info("Starting GradeBlackbox task")

	acquiredSandbox := pool.Pool.Acquire()
	defer pool.Pool.Release(acquiredSandbox)

	log.WithField("boxId", acquiredSandbox.BoxId).Debug("Acquired sandbox")

	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", acquiredSandbox.BoxId)
	if err != nil {
		log.WithError(err).Error("Failed to create isolate sandbox")
		return "", err
	}

	if err := DownloadFileToSandbox(sbx, "grader.zip", graderFilesURL); err != nil {
		log.WithError(err).WithField("file", "grader.zip").Error("Failed to download grader files")
		return "", err
	}
	log.Info("Downloaded grader.zip successfully")

	if err := DownloadFileToSandbox(sbx, "submission.zip", submissionFilesURL); err != nil {
		log.WithError(err).WithField("file", "submission.zip").Error("Failed to download submission files")
		return "", err
	}
	log.Info("Downloaded submission.zip successfully")

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

	log.WithFields(log.Fields{
		"timeLimitMs":   timeLimit,
		"memoryLimitKb": memoryLimit,
	}).Info("Constructed submission object")

	result := orchestrator.GradeBlackboxSubmission(sbx, submission)
	
	log.WithFields(log.Fields{
		"status":       result.Status,
		"is_success":   result.IsSuccess,
		"testcase_len": len(result.TestcaseGradingResult),
	}).Info("Received result from grader")

	log.WithField("full_result", result).Debug("Grading result detail")

	b, err := json.Marshal(result)
	if err != nil {
		log.WithError(err).Error("Failed to marshal grading result")
		return err.Error(), err
	}

	log.Info("Grading completed successfully")
	return string(b), nil
}
