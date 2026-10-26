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

// Limits for the in-box submission unzip. Compile and run set their own limits afterwards.
const (
	unzipTimeLimitMs     = 10000
	unzipWallTimeLimitMs = 20000
	unzipMemoryLimitKb   = 262144
)

// DownloadFileToSandbox downloads an archive into the box and unzips it there.
// Files already in the box (teacher files) are never overwritten. Root entries
// named in remove are deleted after the unzip, whatever their type.
func DownloadFileToSandbox(ctx context.Context, sbx sandboxes.Sandbox, filename, url string, remove ...string) error {
	if err := files.DownloadFile(ctx, sbx.GetBoxdir()+"/"+filename, url); err != nil {
		return err
	}

	red := sandboxes.CreateRedirectionFiles(sbx)
	if err := red.CreateNewMetaFileAndRedirect("_" + files.RemoveExtention(filename) + "_unzip.meta"); err != nil {
		return err
	}

	sbx.SetTimeLimitInMiliseconds(unzipTimeLimitMs)
	sbx.SetWallTimeLimitInMiliseconds(unzipWallTimeLimitMs)
	sbx.SetMemoryLimitInKilobytes(unzipMemoryLimitKb)

	result := sbx.Execute(
		*command.GetCommandBuilder("/usr/bin/unzip").AddArgs("-q", "-n", filename),
		red,
	)

	if result.Status != sandboxes.ZERO_EXIT_CODE {
		return errors.New("Something went wrong while unzipping: " + result.Status.String() + " " + result.Message)
	}

	for _, name := range remove {
		if err := os.RemoveAll(filepath.Join(sbx.GetBoxdir(), name)); err != nil {
			return err
		}
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

func prepareGraderFiles(ctx context.Context, sbx sandboxes.Sandbox, taskDir, url string, inputs, outputs []string) ([]models.Testcase, error) {
	if err := os.RemoveAll(taskDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(taskDir, 0700); err != nil {
		return nil, err
	}
	if err := files.DownloadFile(ctx, filepath.Join(taskDir, "grader.zip"), url); err != nil {
		return nil, err
	}
	unzipCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(unzipCtx, "/usr/bin/unzip", "-q", "-o", "grader.zip", "-d", filepath.Join(taskDir, "files"))
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
		// Hidden inputs are copied by the engine just before their own test runs.
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) != ".out" && filepath.Ext(entry.Name()) != ".in" {
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
