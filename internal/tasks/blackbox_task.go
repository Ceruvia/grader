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
	acquiredSandbox := pool.Pool.Acquire()
	defer pool.Pool.Release(acquiredSandbox)

	log.WithFields(log.Fields{
		"submissionId": submissionId,
		"sandboxId": acquiredSandbox.BoxId,
	}).Debug("Acquired sandbox")

	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", acquiredSandbox.BoxId)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to create isolate sandbox")
		return "", err
	}

	if err := DownloadFileToSandbox(sbx, "grader.zip", graderFilesURL); err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grader files")
		return "", err
	}

	if err := DownloadFileToSandbox(sbx, "submission.zip", submissionFilesURL); err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grader files")
		return "", err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish preparing files")

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
		"submissionId": submissionId,
	}).Debug("Start grading")

	result := orchestrator.GradeBlackboxSubmission(sbx, submission)

	b, err := json.Marshal(result)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to marshal grading result")
		return err.Error(), err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish grading")

	return string(b), nil
}

func GradeBlackboxWithBuilder(
	submissionId string,
	graderFilesURL, submissionFilesURL string,
	inputTestcases, outputTestcases []string,
	timeLimit, memoryLimit int,
	language string,
	builder, compileScript, runScript string,
) (string, error) {
	acquiredSandbox := pool.Pool.Acquire()
	defer pool.Pool.Release(acquiredSandbox)

	log.WithFields(log.Fields{
		"submissionId": submissionId,
		"sandboxId": acquiredSandbox.BoxId,
	}).Debug("Acquired sandbox")

	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", acquiredSandbox.BoxId)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to create isolate sandbox")
		return "", err
	}

	if err := DownloadFileToSandbox(sbx, "grader.zip", graderFilesURL); err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grader files")
		return "", err
	}

	if err := DownloadFileToSandbox(sbx, "submission.zip", submissionFilesURL); err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grader files")
		return "", err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish preparing files")

	submission := &models.SubmissionWithBuilder{
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
		Builder: builder,
		CompileScript: compileScript,
		RunScript: runScript,
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Start grading")

	result := orchestrator.GradeBlackboxSubmission(sbx, submission)

	b, err := json.Marshal(result)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to marshal grading result")
		return err.Error(), err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish grading")

	return string(b), nil
}
