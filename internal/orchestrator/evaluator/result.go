package evaluator

import "github.com/Ceruvia/grader/internal/models"

type EngineRunResult struct {
	Verdict         models.Verdict
	HasErrorMessage bool
	ErrorMessage    string

	InputFilename  string
	OutputFilename string

	TimeToRunInMiliseconds int
	MemoryUsedInKilobytes  int
}

type GradingResult struct {
	IsSuccess             bool
	Status                string
	ErrorMessage          string
	TestcaseGradingResult []EngineRunResult
}
