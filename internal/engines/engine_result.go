package engines

import "github.com/Ceruvia/grader/internal/models"

type EngineRunResult struct {
	Verdict                models.Verdict
	TimeToRunInMiliseconds int
	MemoryUsedInKilobytes  int
	HasErrorMessage        bool
	ErrorMessage           string
}
