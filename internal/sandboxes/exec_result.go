package sandboxes

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type SandboxExecutionResult struct {
	Status     string
	ExitSignal int
	Time       float64
	WallTime   float64
	Memory     int
	Message    string
	IsKilled   bool
}

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
	status := result["status"]
	message := result["message"]
	exitSignal, _ := strconv.Atoi(result["exitsig"])
	isKilledParsed := result["killed"]

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
		Message:    message,
		IsKilled:   isKilled,
	}, nil
}
