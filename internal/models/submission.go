package models

type Submission struct {
	Id         string
	Language   string
	BuildFiles []string
	TCInputFiles []string
	TCOutputFiles []string
}