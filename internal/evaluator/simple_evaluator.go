package evaluator

import (
	"bytes"
	"fmt"

	"github.com/Ceruvia/grader/internal/engines"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type SimpleEvaluator struct{}

func (se SimpleEvaluator) Evaluate(sbx sandboxes.Sandbox, execResult sandboxes.SandboxExecutionResult, expectedOutputFilenameInBox, actualOutputFilenameInBox string) (engines.EngineRunResult, error) {
	finalVerdict := models.VerdictXX

	switch execResult.Status {
	case sandboxes.NONZERO_EXIT_CODE:
		finalVerdict = models.VerdictRE
	case sandboxes.KILLED_ON_SIGNAL:
		finalVerdict = models.VerdictRE
	case sandboxes.TIMED_OUT:
		finalVerdict = models.VerdictTLE
	case sandboxes.INTERNAL_ERROR:
		finalVerdict = models.VerdictXX
	case sandboxes.PARSING_META_ERROR:
		finalVerdict = models.VerdictXX
	}

	expectedOutput, err := sbx.GetFile(expectedOutputFilenameInBox)
	if err != nil {
		return engines.EngineRunResult{
				Verdict:         models.VerdictXX,
				HasErrorMessage: true,
				ErrorMessage:    fmt.Sprintf("Failed to read expected output file", err)},
			err
	}

	actualOutput, err := sbx.GetFile(actualOutputFilenameInBox)
	if err != nil {
		return engines.EngineRunResult{
				Verdict:         models.VerdictXX,
				HasErrorMessage: true,
				ErrorMessage:    fmt.Sprintf("Failed to read actual output file", err)},
			err
	}

	if bytes.Equal(actualOutput, expectedOutput) {
		finalVerdict = models.VerdictAC
	} else {
		finalVerdict = models.VerdictWA
	}

	return engines.EngineRunResult{
		Verdict:                finalVerdict,
		TimeToRunInMiliseconds: int(execResult.Time * 1000),
		MemoryUsedInKilobytes:  execResult.Memory,
		HasErrorMessage:        false,
	}, nil
}
