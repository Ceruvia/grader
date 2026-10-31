package sandboxes

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/Ceruvia/grader/internal/helper/diagnostics"
)

func ParseMetaResult(metaFilePath string) (SandboxExecutionResult, error) {
	file, err := os.Open(metaFilePath)
	if err != nil {
		return SandboxExecutionResult{}, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := parts[0]
			val := parts[1]
			result[key] = val
		}
	}

	if err := scanner.Err(); err != nil {
		return SandboxExecutionResult{}, err
	}

	time, timeErr := strconv.ParseFloat(result["time"], 64)
	wallTime, wallErr := strconv.ParseFloat(result["time-wall"], 64)
	memory, _ := strconv.Atoi(result["cg-mem"])
	statusParsed := result["status"]
	message := diagnostics.Limit(result["message"])
	exitSignal, _ := strconv.Atoi(result["exitsig"])
	exitCode, exitErr := strconv.Atoi(result["exitcode"])

	isKilledParsed := result["killed"]

	status := ZERO_EXIT_CODE
	switch statusParsed {
	case "RE":
		status = NONZERO_EXIT_CODE
	case "SG":
		status = KILLED_ON_SIGNAL
	case "TO":
		status = TIMED_OUT
	case "XX":
		status = INTERNAL_ERROR
	case "":
		// Isolate omits status on success, but always writes exitcode and times.
		if exitErr != nil || exitCode != 0 || timeErr != nil || wallErr != nil {
			return SandboxExecutionResult{}, fmt.Errorf("incomplete isolate success metadata")
		}
	default:
		return SandboxExecutionResult{}, fmt.Errorf("unknown isolate status %q", statusParsed)
	}
	if _, present := result["time"]; present && (timeErr != nil || math.IsNaN(time) || math.IsInf(time, 0) || time < 0) {
		return SandboxExecutionResult{}, fmt.Errorf("invalid isolate time")
	}
	if _, present := result["time-wall"]; present && (wallErr != nil || math.IsNaN(wallTime) || math.IsInf(wallTime, 0) || wallTime < 0) {
		return SandboxExecutionResult{}, fmt.Errorf("invalid isolate wall time")
	}

	isKilled := false
	if isKilledParsed == "1" {
		isKilled = true
	}

	return SandboxExecutionResult{
		Time:       time,
		WallTime:   wallTime,
		Memory:     memory,
		Status:     status,
		ExitSignal: exitSignal,
		ExitCode:   exitCode,
		Message:    message,
		IsKilled:   isKilled,
	}, nil
}
