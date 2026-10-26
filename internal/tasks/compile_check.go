package tasks

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/compilers"
	log "github.com/sirupsen/logrus"
)

const compileMessageLimit = 64 << 10

type compileCheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func CompileCheck(submissionId, graderFilesURL, submissionFilesURL string, language, mainSourceFilename string) (string, error) {
	return compileCheck(graderFilesURL, submissionFilesURL, &models.SubmissionWithFiles{
		Core: models.Core{Id: submissionId, Language: language}, MainSourceFilename: mainSourceFilename,
	})
}

func CompileCheckWithBuilder(submissionId, graderFilesURL, submissionFilesURL string, language, builder, compileScript string) (string, error) {
	return compileCheck(graderFilesURL, submissionFilesURL, &models.SubmissionWithBuilder{
		Core: models.Core{Id: submissionId, Language: language}, Builder: builder, CompileScript: compileScript,
	})
}

func compileCheck(graderFilesURL, submissionFilesURL string, submission models.Submission) (string, error) {
	sbx, _, cleanup, err := prepareTask(submission.GetId(), graderFilesURL, submissionFilesURL, nil, nil)
	if err != nil {
		log.WithField("submission_id", submission.GetId()).WithError(err).Error("Failed to prepare compile check")
		return encodeCompileCheck("internal_error", "Test could not run. Please try again.")
	}
	defer cleanup()

	compiled, err := compilers.CompileSubmission(sbx, submission, compileMessageLimit)
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
	// JSON replaces invalid UTF-8 with a three-byte rune. Normalize before
	// truncation so the decoded message, not just the raw compiler bytes, is bounded.
	message = strings.ToValidUTF8(message, "\ufffd")
	if len(message) > compileMessageLimit {
		end := compileMessageLimit
		for !utf8.RuneStart(message[end]) {
			end--
		}
		message = message[:end]
	}
	b, err := json.Marshal(compileCheckResult{Status: status, Message: message})
	return string(b), err
}
