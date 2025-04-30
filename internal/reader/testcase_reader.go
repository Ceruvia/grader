package reader

import (
	"errors"
	"io/fs"
	"path/filepath"
)

var (
	ErrInputNotExist  = errors.New("input file does not exist")
	ErrOutputNotExist = errors.New("output file does not exist")
)

// Reads the input and output file concurrently using filesystem and relative workdir
// to return the input data and output data.
func ReadTestcaseInputOutputFile(fsys fs.FS, workdir, inputFilename, outputFilename string) (string, string, error) {
	inputFilepath := filepath.Join(workdir, inputFilename)
	outputFilepath := filepath.Join(workdir, outputFilename)

	inputData, inputErr := fs.ReadFile(fsys, inputFilepath)

	if inputErr != nil {
		return "", "", ErrInputNotExist
	}

	outputData, outputErr := fs.ReadFile(fsys, outputFilepath)

	if outputErr != nil {
		return "", "", ErrOutputNotExist
	}

	return string(inputData), string(outputData), nil
}

// Reads the input and output file concurrently using filesystem and relative workdir
// to return the input data and output data.
// This uses GOROUTINES.
// func ReadTestcaseInputOutputFile(fsys fs.FS, workdir, inputFilename, outputFilename string) (string, string, error) {
// 	inputFilepath := filepath.Join(workdir, inputFilename)
// 	outputFilepath := filepath.Join(workdir, outputFilename)

// 	var (
// 		inputData, outputData []byte
// 		inputErr, outputErr   error
// 	)

// 	var wg sync.WaitGroup

// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		inputData, inputErr = fs.ReadFile(fsys, inputFilepath)
// 	}()

// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		outputData, outputErr = fs.ReadFile(fsys, outputFilepath)
// 	}()

// 	wg.Wait()

// 	if inputErr != nil {
// 		return "", "", ErrInputNotExist
// 	}

// 	if outputErr != nil {
// 		return "", "", ErrOutputNotExist
// 	}

// 	return string(inputData), string(outputData), nil
// }
