package tasks

import (
	"testing"

	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

const (
	SAMPLE_SUBMISSION_FILE_URL = "https://pub-aa14e9fb26a94974a23c01cf74108727.r2.dev/c_adt_submission.zip"
	SAMPLE_GRADER_FILE_URL     = "https://pub-aa14e9fb26a94974a23c01cf74108727.r2.dev/c_adt_grader.zip"
	SAMPLE_WRONG_FILE_URL      = "https://pub-aa14e9fb26a94974a23c01cf74108727.r2.dev/gaada.zip"
)

func TestPrepareHelpers(t *testing.T) {
	t.Run("DownloadFilesToSandbox", func(t *testing.T) {
		t.Run("Sandbox populated by unzipped files", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 400)

			sbx.SetTimeLimitInMiliseconds(20 * 1000)     // 20 seconds
			sbx.SetWallTimeLimitInMiliseconds(60 * 1000) // 1 minute
			sbx.SetMemoryLimitInKilobytes(1024 * 1024)   // 1 GB

			tester.AssertNotError(t, err)

			err = DownloadFileToSandbox(sbx, "submission.zip", SAMPLE_SUBMISSION_FILE_URL)
			tester.AssertNotError(t, err)

			err = DownloadFileToSandbox(sbx, "grader.zip", SAMPLE_GRADER_FILE_URL)
			tester.AssertNotError(t, err)

			got := sbx.Filenames
			want := []string{"_submission_unzip.meta", "ganjilgenap.c", "submission.zip", "1.in", "1.out", "10.in", "10.out", "2.in", "2.out", "3.in", "3.out", "4.in", "4.out", "5.in", "5.out", "6.in", "6.out", "7.in", "7.out", "8.in", "8.out", "9.in", "9.out", "_grader_unzip.meta", "_submission_unzip.meta", "array.c", "array.h", "boolean.h", "ganjilgenap.c", "grader.zip", "submission.zip"}

			tester.AssertDeep(t, got, want)
		})

		t.Run("Returns error when link isn't a ZIP file", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 401)

			sbx.SetTimeLimitInMiliseconds(20 * 1000)     // 20 seconds
			sbx.SetWallTimeLimitInMiliseconds(60 * 1000) // 1 minute
			sbx.SetMemoryLimitInKilobytes(1024 * 1024)   // 1 GB

			tester.AssertNotError(t, err)

			err = DownloadFileToSandbox(sbx, "submission.zip", SAMPLE_WRONG_FILE_URL)
			if err == nil {
				t.Errorf("Expected an error but got Nil instead")
			}
		})
	})
}
