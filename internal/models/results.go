package models

// Sandbox execution results

// Compiler compilation execution results


// Engine grading results
type EngineRunResult struct {
	Verdict         Verdict
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
