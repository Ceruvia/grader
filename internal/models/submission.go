package models

type Submission struct {
	Language  string
	Testcases []Testcase
}

type Testcase struct {
	Input          string
	ExpectedOutput string
}
