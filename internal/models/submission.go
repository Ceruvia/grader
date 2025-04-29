package models

type Submission struct {
	Language   string
	BuildFiles []string
	Testcases  []Testcase

	Workdir             string
	HasCompletedGrading bool
	TimeStarted         string
	TimeCompleted       string
	FinalScore          float32
}

type Testcase struct {
	InputFilename  string
	OutputFilename string
}
