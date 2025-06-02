package models

import (
	"github.com/Ceruvia/grader/internal/languages"
)

type Submission interface {
	IsBuilder() bool

	GetId() string
	GetEngine() string
	GetLanguage() languages.Language
	GetCompileLanguage() languages.Language
	GetLimits() GradingLimit
	GetTestcases() []Testcase

	GetExecFilenameOrScript() string
	GetCompileFilenameOrScript() string
}

type SubmissionWithBuilder struct {
	Core
	Builder       string
	CompileScript string
	RunScript     string
}

func (s SubmissionWithBuilder) IsBuilder() bool   { return true }
func (s SubmissionWithBuilder) GetId() string     { return s.Core.Id }
func (s SubmissionWithBuilder) GetEngine() string { return s.Core.Engine }
func (s SubmissionWithBuilder) GetLanguage() languages.Language {
	return languages.GetLanguage(s.Core.Language)
}
func (s SubmissionWithBuilder) GetCompileLanguage() languages.Language {
	return languages.GetLanguage(s.Builder)
}
func (s SubmissionWithBuilder) GetLimits() GradingLimit            { return s.Core.Limits }
func (s SubmissionWithBuilder) GetTestcases() []Testcase           { return s.Core.Testcases }
func (s SubmissionWithBuilder) GetExecFilenameOrScript() string    { return s.RunScript }
func (s SubmissionWithBuilder) GetCompileFilenameOrScript() string { return s.CompileScript }

type SubmissionWithFiles struct {
	Core
	MainSourceFilename string
}

func (s SubmissionWithFiles) IsBuilder() bool   { return false }
func (s SubmissionWithFiles) GetId() string     { return s.Core.Id }
func (s SubmissionWithFiles) GetEngine() string { return s.Core.Engine }
func (s SubmissionWithFiles) GetLanguage() languages.Language {
	return languages.GetLanguage(s.Core.Language)
}
func (s SubmissionWithFiles) GetCompileLanguage() languages.Language {
	return languages.GetLanguage(s.Core.Language)
}
func (s SubmissionWithFiles) GetLimits() GradingLimit  { return s.Core.Limits }
func (s SubmissionWithFiles) GetTestcases() []Testcase { return s.Core.Testcases }
func (s SubmissionWithFiles) GetExecFilenameOrScript() string {
	return s.GetLanguage().GetExecutableFilename(s.MainSourceFilename)
}
func (s SubmissionWithFiles) GetCompileFilenameOrScript() string { return s.MainSourceFilename }

type Core struct {
	Id        string
	Engine    string
	Language  string
	Limits    GradingLimit
	Testcases []Testcase
}
type GradingLimit struct {
	TimeInMiliseconds int
	MemoryInKilobytes int
}
type Testcase struct {
	InputFilename  string
	OutputFilename string
}
