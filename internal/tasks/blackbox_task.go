package tasks

import (
	"encoding/json"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
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
	task, err := prepareTask(submissionId, graderFilesURL, submissionFilesURL, inputTestcases, outputTestcases)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grading task")
		return "", err
	}
	defer task.cleanup()

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
			Testcases: task.testcases,
		},
		MainSourceFilename: mainSourceFilename,
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Start grading")

	result := orchestrator.GradeBlackboxSubmission(task.sbx, submission)

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
	task, err := prepareTask(submissionId, graderFilesURL, submissionFilesURL, inputTestcases, outputTestcases)
	if err != nil {
		log.WithField("submissionId", submissionId).WithError(err).Error("Failed to prepare grading task")
		return "", err
	}
	defer task.cleanup()

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
			Testcases: task.testcases,
		},
		Builder:       builder,
		CompileScript: compileScript,
		RunScript:     runScript,
		Makefile:      task.makefile,
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Start grading")

	result := orchestrator.GradeBlackboxSubmission(task.sbx, submission)

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
