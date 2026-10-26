package tasks

import (
	"context"
	"fmt"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
	log "github.com/sirupsen/logrus"
)

// prepareTask owns the pool lease, host task directory and isolate sandbox together.
// The caller must defer cleanup until it has consumed the compiler/grading result.
func prepareTask(submissionId, graderFilesURL, submissionFilesURL string, inputs, outputs []string) (*sandboxes.IsolateSandbox, []models.Testcase, func(), error) {
	lease := pool.Pool.Acquire()
	log.WithFields(log.Fields{"submissionId": submissionId, "sandboxId": lease.BoxId}).Debug("Acquired sandbox")
	sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", lease.BoxId)
	if err != nil {
		pool.Pool.Release(lease)
		return nil, nil, nil, err
	}
	cleanup := func() {
		defer pool.Pool.Release(lease)
		_ = sbx.Cleanup()
	}
	// Machinery does not give tasks a cancellable context. Download, host unzip,
	// sandbox unzip and compiler limits bound each operation independently.
	ctx := context.Background()
	testcases, err := prepareGraderFiles(ctx, sbx, sandboxes.TaskDir(lease.BoxId), graderFilesURL, inputs, outputs)
	if err == nil {
		err = DownloadFileToSandbox(ctx, sbx, "submission.zip", submissionFilesURL)
	}
	if err != nil {
		cleanup()
		return nil, nil, nil, fmt.Errorf("prepare task: %w", err)
	}
	return sbx, testcases, cleanup, nil
}
