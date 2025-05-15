package models

type Submission struct {
	// Meta descriptions
	Id       string
	TempDir  string
	Language string
	Limits   GradingLimit

	// Builder related stuff (example: Makefile)
	UseBuilder    bool
	Builder       string
	BuildScript   string
	CompileScript string

	// Non-builder related stuff
	MainSourceFilename string

	// Testcases
	TCInputFiles  []string
	TCOutputFiles []string
}

type GradingLimit struct {
	TimeInMiliseconds int
	MemoryInKilobytes int
}
