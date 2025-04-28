package models

type Submission struct {
	Language   string
	UseBuilder bool
	Builder    string
	Files      []string
	Testcases  []Testcase

	Workdir             string
	HasCompletedGrading bool
	TimeStarted         string
	TimeCompleted       string
	FinalScore          float32
}

type Testcase struct {
	InputFilename          string
	ExpectedOutputFilename string
	ActualOutputFilename   string
	Verdict                string
}
