package evaluator

import (
	"bytes"
	"fmt"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

type SimpleEvaluator struct{}

func (se SimpleEvaluator) Evaluate(sbx sandboxes.Sandbox, execResult sandboxes.SandboxExecutionResult, expectedOutputFilenameInBox, actualOutputFilenameInBox string) EngineRunResult {
	intermediateVerdict := models.VerdictWA
	switch execResult.Status {
	case sandboxes.NONZERO_EXIT_CODE:
		intermediateVerdict = models.VerdictRE
	case sandboxes.KILLED_ON_SIGNAL:
		intermediateVerdict = models.VerdictRE
	case sandboxes.TIMED_OUT:
		intermediateVerdict = models.VerdictTLE
	case sandboxes.INTERNAL_ERROR:
		intermediateVerdict = models.VerdictXX
	case sandboxes.PARSING_META_ERROR:
		intermediateVerdict = models.VerdictXX
	}
	actualOutput, err := sbx.GetFile(actualOutputFilenameInBox)
	if err != nil {
		return EngineRunResult{
			Verdict:         models.VerdictXX,
			HasErrorMessage: true,
			ErrorMessage:    fmt.Sprintf("Failed to read actual output file: %q", err)}
	}

	// If intermediate verdict has changed (one of the above status) then instantly return with error from stderr
	if intermediateVerdict != models.VerdictWA {
		return EngineRunResult{
			Verdict:         intermediateVerdict,
			HasErrorMessage: true,
			ErrorMessage:    string(actualOutput)}
	}

	expectedOutput, err := sbx.GetFile(expectedOutputFilenameInBox)
	if err != nil {
		return EngineRunResult{
			Verdict:         models.VerdictXX,
			HasErrorMessage: true,
			ErrorMessage:    fmt.Sprintf("Failed to read expected output file: %q", err)}
	}

	finalVerdict := models.VerdictXX
	if bytes.Equal(
		normalizeNewLines(actualOutput),
		normalizeNewLines(expectedOutput),
	) {
		finalVerdict = models.VerdictAC
	} else {
		finalVerdict = models.VerdictWA
	}

	return EngineRunResult{
		Verdict:                finalVerdict,
		TimeToRunInMiliseconds: int(execResult.Time * 1000),
		MemoryUsedInKilobytes:  execResult.Memory,
		HasErrorMessage:        false,
	}
}

func normalizeNewLines(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
