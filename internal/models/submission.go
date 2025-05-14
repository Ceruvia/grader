package models

type Submission struct {
	Id             string
	TempDir        string
	Language       string
	BuildFiles     []string // files originating from problem statement
	SubmittedFiles []string // files originating from user upload / submit
	TCInputFiles   []string
	TCOutputFiles  []string
	Limits         GradingLimit
}

type GradingLimit struct {
	TimeInMiliseconds int
	MemoryInKilobytes int
}
