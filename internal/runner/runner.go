package runner

import (
	"errors"
	"os"
	"os/exec"
)

func Run(binaryFilename string) (string, error) {
	if _, err := os.Stat(binaryFilename); errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	cmd := exec.Command(binaryFilename)

	result, err := cmd.CombinedOutput()

	if _, ok := err.(*exec.ExitError); ok {
		return string(result), nil
	} else if err != nil {
		return "", err
	}

	return string(result), nil
}

func RunWithInputAndOutput(binaryFilename, inputFilePath, outputFilePath string) (bool, error) {
	if _, err := os.Stat(binaryFilename); errors.Is(err, os.ErrNotExist) {
		return false, err
	}

	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return false, err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return false, err
	}
	defer outputFile.Close()

	cmd := exec.Command(binaryFilename)
	cmd.Stdin = inputFile
	cmd.Stdout = outputFile
	err = cmd.Run()

	if _, ok := err.(*exec.ExitError); ok {
		return true, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func Grade(expectedOutputFilePath, actualOutputFilePath string) (bool, error) {
	expected, err := ReadFile(expectedOutputFilePath)
	if err != nil {
		return false, err
	}

	actual, err := ReadFile(actualOutputFilePath)
	if err != nil {
		return false, err
	}

	return expected == actual, nil
}

func ReadFile(filePath string) (string, error) {
	dat, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(dat), nil
}
