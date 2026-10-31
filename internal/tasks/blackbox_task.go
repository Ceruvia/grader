package tasks

import (
	"encoding/json"
	"errors"

	"github.com/Ceruvia/grader/internal/helper/diagnostics"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
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
		return "", errors.New(diagnostics.Limit(err.Error()))
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

	encoded, err := encodeGradingResult(submissionId, result)
	if err != nil {
		return "", err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish grading")

	return encoded, nil
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
		return "", errors.New(diagnostics.Limit(err.Error()))
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

	encoded, err := encodeGradingResult(submissionId, result)
	if err != nil {
		return "", err
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
	}).Debug("Finish grading")

	return encoded, nil
}

func encodeGradingResult(submissionId string, result evaluator.GradingResult) (string, error) {
	if !result.IsSuccess && result.Status == orchestrator.StatusInternalError {
		log.WithField("submissionId", submissionId).WithField("detail", diagnostics.Limit(result.ErrorMessage)).Error("Grading infrastructure failed")
		// Backend maps Machinery FAILURE to XX. A successful task with
		// is_success=false would instead blame the student with CE.
		return "", errors.New("Internal grader error. Please contact course staff.")
	}
	// Bound diagnostics in aggregate as well as individually, so many failing
	// cases cannot multiply 64 KiB into an oversized result-backend message.
	remaining := diagnostics.MaxBytes
	result.ErrorMessage = diagnostics.Prefix(result.ErrorMessage, remaining)
	remaining -= len(result.ErrorMessage)
	for i := range result.TestcaseGradingResult {
		message := &result.TestcaseGradingResult[i].ErrorMessage
		*message = diagnostics.Prefix(*message, remaining)
		remaining -= len(*message)
	}
	b, err := json.Marshal(result)
	return string(b), err
}
