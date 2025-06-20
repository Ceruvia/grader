package sandboxes

import (
	"bufio"
	"os"
	"strconv"
	"strings"
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

	time, _ := strconv.ParseFloat(result["time"], 64)
	wallTime, _ := strconv.ParseFloat(result["time-wall"], 64)
	memory, _ := strconv.Atoi(result["cg-mem"])
	statusParsed := result["status"]
	message := result["message"]
	exitSignal, _ := strconv.Atoi(result["exitsig"])
	exitCode, _ := strconv.Atoi(result["exitcode"])

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
	default:
		status = ZERO_EXIT_CODE
	}

	isKilled := false
	if isKilledParsed == "1" {
		isKilled = true
	}

	return SandboxExecutionResult{
		Time:       time * 1000,
		WallTime:   wallTime * 1000,
		Memory:     memory,
		Status:     status,
		ExitSignal: exitSignal,
		ExitCode:   exitCode,
		Message:    message,
		IsKilled:   isKilled,
	}, nil
}
