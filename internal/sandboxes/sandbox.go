package sandboxes

type Sandbox interface {
	AddFile()
	ContainsFile()
	GetFile()
	AddAllowedDirectory()
	SetTimeLimitInMiliseconds()
	SetMemoryLimitInKilobytes()
	ResetRedirections()
	RedirectStandardInput()
	RedirectStandardOutput()
	RedirectStandardError()
	CleanUp()

	Execute()
	GetResult()
}