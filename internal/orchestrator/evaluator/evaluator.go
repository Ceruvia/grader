package evaluator

import (
	"github.com/Ceruvia/grader/internal/sandboxes"
)

type Evaluator interface {
	Evaluate(sbx sandboxes.Sandbox, execResult sandboxes.SandboxExecutionResult, expectedOutputPath, actualOutputFilenameInBox string, messageLimit int) EngineRunResult
}
