package sandboxes

type Sandbox interface {
	AddFile()
	ContainsFile()
	GetFile()
	AddAllowedDirectory()
	SetTimeLimitInMiliseconds()
	SetWallTimeLimitInMiliseconds()
	SetMemoryLimitInKilobytes()
	ResetRedirection()
	RedirectStandardInput()
	RedirectStandardOutput()
	RedirectStandardError()
	CleanUp()

	Execute()
	GetResult()
}