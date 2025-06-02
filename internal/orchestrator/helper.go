package orchestrator

import "github.com/Ceruvia/grader/internal/orchestrator/evaluator"

func createFailGradingResult(status, errorMessage string) evaluator.GradingResult {
	return evaluator.GradingResult{
		Status:       status,
		IsSuccess:    false,
		ErrorMessage: errorMessage,
	}
}
