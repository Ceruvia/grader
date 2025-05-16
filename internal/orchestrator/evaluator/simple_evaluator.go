package evaluator

import (
	"bytes"
	"fmt"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

type SimpleEvaluator struct{}

func (se SimpleEvaluator) Evaluate(sbx sandboxes.Sandbox, execResult sandboxes.SandboxExecutionResult, expectedOutputFilenameInBox, actualOutputFilenameInBox string) (models.EngineRunResult, error) {
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
		return models.EngineRunResult{
			Verdict:         models.VerdictXX,
			HasErrorMessage: true,
			ErrorMessage:    fmt.Sprintf("Failed to read actual output file: %q", err)}, nil
	}
	if intermediateVerdict != models.VerdictWA { // If intermediate verdict has changed (one of the above status) then instantly return with error from stderr
		return models.EngineRunResult{
			Verdict:         intermediateVerdict,
			HasErrorMessage: true,
			ErrorMessage:    string(actualOutput)}, nil
	}

	expectedOutput, err := sbx.GetFile(expectedOutputFilenameInBox)
	if err != nil {
		return models.EngineRunResult{
			Verdict:         models.VerdictXX,
			HasErrorMessage: true,
			ErrorMessage:    fmt.Sprintf("Failed to read expected output file: %q", err)}, nil
	}

	finalVerdict := models.VerdictXX
	if bytes.Equal(actualOutput, expectedOutput) {
		finalVerdict = models.VerdictAC
	} else {
		finalVerdict = models.VerdictWA
	}

	return models.EngineRunResult{
		Verdict:                finalVerdict,
		TimeToRunInMiliseconds: int(execResult.Time * 1000),
		MemoryUsedInKilobytes:  execResult.Memory,
		HasErrorMessage:        false,
	}, nil
}
