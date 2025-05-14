package models

type EngineRunResult struct {
	Verdict                Verdict
	TimeToRunInMiliseconds int
	MemoryUsedInKilobytes  int
	HasErrorMessage        bool
	ErrorMessage           string
}
