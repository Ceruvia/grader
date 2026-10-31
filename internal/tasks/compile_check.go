package tasks

import (
	"encoding/json"
	"time"

	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	log "github.com/sirupsen/logrus"
)

const compileMessageLimit = diagnostics.MaxBytes

var compileCheckNow = time.Now

type compileCheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func CompileCheck(submissionId, graderFilesURL, submissionFilesURL string, language, mainSourceFilename string, deadlineUnixMs int64) (string, error) {
	return compileCheck(graderFilesURL, submissionFilesURL, &models.SubmissionWithFiles{
		Core: models.Core{Id: submissionId, Language: language}, MainSourceFilename: mainSourceFilename,
	}, deadlineUnixMs)
}

func CompileCheckWithBuilder(submissionId, graderFilesURL, submissionFilesURL string, language, builder, compileScript string, deadlineUnixMs int64) (string, error) {
	return compileCheck(graderFilesURL, submissionFilesURL, &models.SubmissionWithBuilder{
		Core: models.Core{Id: submissionId, Language: language}, Builder: builder, CompileScript: compileScript,
	}, deadlineUnixMs)
}

func compileCheck(graderFilesURL, submissionFilesURL string, submission models.Submission, deadlineUnixMs int64) (string, error) {
	if now := compileCheckNow().UnixMilli(); deadlineUnixMs > 0 && now > deadlineUnixMs {
		log.WithFields(log.Fields{"submissionId": submission.GetId(), "late_ms": now - deadlineUnixMs}).Info("Skipping expired compile check")
		return encodeCompileCheck("internal_error", "Test could not run. Please try again.")
	}
	task, err := prepareTask(submission.GetId(), graderFilesURL, submissionFilesURL, nil, nil)
	if err != nil {
		log.WithField("submission_id", submission.GetId()).WithError(err).Error("Failed to prepare compile check")
		return encodeCompileCheck("internal_error", "Test could not run. Please try again.")
	}
	defer task.cleanup()
	if builder, ok := submission.(*models.SubmissionWithBuilder); ok {
		builder.Makefile = task.makefile
	}

	compiled, err := compilers.CompileSubmission(task.sbx, submission, compileMessageLimit)
	if err != nil {
		log.WithField("submission_id", submission.GetId()).WithError(err).Error("Failed to prepare compiler")
		return encodeCompileCheck("internal_error", "Test could not run. Please try again.")
	}
	if compiled.InternalError {
		log.WithField("submission_id", submission.GetId()).WithField("detail", compiled.StdoutStderr).Error("Compile check internal error")
		return encodeCompileCheck("internal_error", "Test could not run. Please try again.")
	}
	if !compiled.IsSuccess {
		return encodeCompileCheck("compile_error", compiled.StdoutStderr)
	}
	return encodeCompileCheck("compiled", "")
}

func encodeCompileCheck(status, message string) (string, error) {
	b, err := json.Marshal(compileCheckResult{Status: status, Message: diagnostics.Limit(message)})
	return string(b), err
}
