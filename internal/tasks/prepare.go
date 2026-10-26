package tasks

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/files"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

func DownloadFileToSandbox(sbx sandboxes.Sandbox, filename, url string) error {
	if err := files.DownloadFile(sbx.GetBoxdir()+"/"+filename, url); err != nil {
		return err
	}

	red := sandboxes.CreateRedirectionFiles(sbx)
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

func prepareGraderFiles(sbx sandboxes.Sandbox, taskDir, url string, inputs, outputs []string) ([]models.Testcase, error) {
	if err := os.RemoveAll(taskDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(taskDir, 0700); err != nil {
		return nil, err
	}
	if err := files.DownloadFile(filepath.Join(taskDir, "grader.zip"), url); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/unzip", "-q", "-o", "grader.zip", "-d", filepath.Join(taskDir, "files"))
	cmd.Dir = taskDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("unzip grader files: %w: %s", err, output)
	}
	graderDir := filepath.Join(taskDir, "files")
	entries, err := os.ReadDir(graderDir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) != ".out" {
			if err := sbx.AddFile(filepath.Join(graderDir, entry.Name())); err != nil {
				return nil, err
			}
		}
	}
	if len(inputs) != len(outputs) {
		return nil, errors.New("testcase input and output counts differ")
	}
	testcases := make([]models.Testcase, 0, len(inputs))
	for i, input := range inputs {
		output := outputs[i]
		if filepath.Base(input) != input || filepath.Ext(input) != ".in" || output != strings.TrimSuffix(input, ".in")+".out" {
			return nil, fmt.Errorf("invalid testcase pair %q, %q", input, output)
		}
		for _, name := range []string{input, output} {
			info, err := os.Lstat(filepath.Join(graderDir, name))
			if err != nil {
				return nil, err
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("testcase file %q is not regular", name)
			}
		}
		testcases = append(testcases, models.Testcase{
			InputFilename:      input,
			OutputFilename:     output,
			ExpectedOutputPath: filepath.Join(graderDir, output),
		})
	}
	return testcases, nil
}
