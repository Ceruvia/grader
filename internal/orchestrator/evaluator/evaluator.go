package evaluator

import (
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

type Evaluator interface {
	Evaluate(sbx sandboxes.Sandbox, execResult models.SandboxExecutionResult, expectedOutputFilenameInBox, actualOutputFilenameInBox string) (models.EngineRunResult, error)
}
