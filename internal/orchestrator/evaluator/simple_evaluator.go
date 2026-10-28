package evaluator

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/Ceruvia/grader/internal/helper/diagnostics"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type SimpleEvaluator struct{}

func (se SimpleEvaluator) Evaluate(sbx sandboxes.Sandbox, execResult sandboxes.SandboxExecutionResult, expectedOutputPath, actualOutputFilenameInBox string, messageLimit int) EngineRunResult {
	messageLimit = max(0, min(messageLimit, diagnostics.MaxBytes))
	switch execResult.Status {
	case sandboxes.INTERNAL_ERROR, sandboxes.PARSING_META_ERROR:
		return runFailure(models.VerdictXX, execResult.Message, messageLimit)
	case sandboxes.NONZERO_EXIT_CODE, sandboxes.KILLED_ON_SIGNAL, sandboxes.TIMED_OUT:
		verdict := models.VerdictRE
		if execResult.Status == sandboxes.TIMED_OUT {
			verdict = models.VerdictTLE
		}
		if messageLimit == 0 {
			// Keep the verdict, but never open, read or allocate another diagnostic
			// once earlier test cases exhausted the per-task budget.
			return runFailure(verdict, "", 0)
		}
		output, err := sandboxes.ReadBoxFile(sbx, actualOutputFilenameInBox, int64(messageLimit))
		if err != nil {
			return runError(verdict, messageLimit, "Failed to read actual output file: %q", err)
		}
		return runFailure(verdict, string(output), messageLimit)
	case sandboxes.ZERO_EXIT_CODE:
		// Compare incrementally. Successful output is never copied into a message.
	default:
		return runFailure(models.VerdictXX, "Unknown sandbox execution status", messageLimit)
	}
	actual, err := sandboxes.OpenBoxFile(sbx, actualOutputFilenameInBox)
	if err != nil {
		return runError(models.VerdictRE, messageLimit, "Failed to read actual output file: %q", err)
	}
	defer actual.Close()
	expected, err := os.Open(expectedOutputPath)
	if err != nil {
		return runError(models.VerdictXX, messageLimit, "Failed to read expected output file: %q", err)
	}
	defer expected.Close()
	equal, err := outputsEqual(actual, expected)
	if err != nil {
		return runError(models.VerdictXX, messageLimit, "Failed to compare output: %q", err)
	}
	verdict := models.VerdictWA
	if equal {
		verdict = models.VerdictAC
	}
	return EngineRunResult{
		Verdict:                 verdict,
		TimeToRunInMilliseconds: int(execResult.Time * 1000),
		MemoryUsedInKilobytes:   execResult.Memory,
	}
}

func runFailure(verdict models.Verdict, message string, messageLimit int) EngineRunResult {
	return EngineRunResult{Verdict: verdict, HasErrorMessage: true, ErrorMessage: diagnostics.Prefix(message, messageLimit)}
}

func runError(verdict models.Verdict, messageLimit int, format string, err error) EngineRunResult {
	if messageLimit <= 0 {
		return runFailure(verdict, "", 0)
	}
	return runFailure(verdict, fmt.Sprintf(format, err), messageLimit)
}

func outputsEqual(actual, expected io.Reader) (bool, error) {
	a, b := bufio.NewReader(actual), bufio.NewReader(expected)
	for {
		ac, ae := normalizedByte(a)
		bc, be := normalizedByte(b)
		if ae != nil && ae != io.EOF {
			return false, ae
		}
		if be != nil && be != io.EOF {
			return false, be
		}
		if ae == io.EOF || be == io.EOF {
			return ae == io.EOF && be == io.EOF, nil
		}
		if ac != bc {
			return false, nil
		}
	}
}

// Preserve the existing CRLF-to-LF normalization, including across buffer edges.
func normalizedByte(r *bufio.Reader) (byte, error) {
	c, err := r.ReadByte()
	if err == nil && c == '\r' {
		if next, _ := r.Peek(1); len(next) == 1 && next[0] == '\n' {
			return r.ReadByte()
		}
	}
	return c, err
}
