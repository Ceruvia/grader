package evaluator

import "github.com/Ceruvia/grader/internal/models"

type EngineRunResult struct {
	Verdict                 models.Verdict `json:"verdict"`
	HasErrorMessage         bool           `json:"has_error_message"`
	ErrorMessage            string         `json:"error_message"`
	InputFilename           string         `json:"input_filename"`
	OutputFilename          string         `json:"output_filename"`
	TimeToRunInMilliseconds int            `json:"time_to_run_ms"`
	MemoryUsedInKilobytes   int            `json:"memory_used_kb"`
}

type GradingResult struct {
	IsSuccess             bool              `json:"is_success"`
	Status                string            `json:"status"`
	ErrorMessage          string            `json:"error_message"`
	TestcaseGradingResult []EngineRunResult `json:"testcase_result"`
}
