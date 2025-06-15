package tasks

import (
	"errors"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func DownloadFileToSandbox(sbx sandboxes.Sandbox, filename, url string) error {
	if err := files.DownloadFile(sbx.GetBoxdir()+"/"+filename, url); err != nil {
		return err
	}

	red := sandboxes.CreateRedirectionFiles(sbx.GetBoxdir())
	red.CreateNewMetaFileAndRedirect("_" + files.RemoveExtention(filename) + "_unzip.meta")

	result := sbx.Execute(
		*command.GetCommandBuilder("/usr/bin/unzip").AddArgs("-q").AddArgs(filename),
		red,
	)

	if result.Status != sandboxes.ZERO_EXIT_CODE {
		return errors.New("Something went wrong while unzipping: " + result.Status.String() + " " + result.Message)
	}

	filenames, err := files.GetFilenamesInDir(sbx.GetBoxdir())
	if err != nil {
		return err
	}

	for _, filename := range filenames {
		sbx.AddFileWithoutMove(filename)
	}

	return nil
}

func createTestcases(inputFilenames, outputFilenames []string) []models.Testcase {
	var testcases []models.Testcase
	for i, inputName := range inputFilenames {
		testcases = append(testcases, models.Testcase{
			InputFilename:  inputName,
			OutputFilename: outputFilenames[i],
		})
	}
	return testcases
}
