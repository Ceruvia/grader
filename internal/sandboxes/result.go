package sandboxes

type SandboxExecutionResult struct {
	Status     SandboxExecutionStatus
	ExitSignal int
	ExitCode   int
	Time       float64
	WallTime   float64
	Memory     int
	Message    string
	IsKilled   bool
}

type SandboxExecutionStatus int

const (
	ZERO_EXIT_CODE SandboxExecutionStatus = iota
	NONZERO_EXIT_CODE
	KILLED_ON_SIGNAL
	TIMED_OUT
	INTERNAL_ERROR
	PARSING_META_ERROR
)

var sandboxExecutionStatusNames = map[SandboxExecutionStatus]string{
	ZERO_EXIT_CODE:     "Success",
	NONZERO_EXIT_CODE:  "Runtime error",
	KILLED_ON_SIGNAL:   "Killed on signal",
	TIMED_OUT:          "Time limit exceeded",
	INTERNAL_ERROR:     "Isolate internal error",
	PARSING_META_ERROR: "Failed to parse meta file",
}

func (s SandboxExecutionStatus) String() string {
	return sandboxExecutionStatusNames[s]
}