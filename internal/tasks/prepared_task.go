package tasks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
	log "github.com/sirupsen/logrus"
)

// makefileNames lists the names GNU make reads without -f, in its own order.
var makefileNames = []string{"GNUmakefile", "makefile", "Makefile"}

// preparedTask owns the pool lease, host task directory and isolate sandbox together.
// The caller must defer cleanup until it has consumed the compiler/grading result.
type preparedTask struct {
	sbx       *sandboxes.IsolateSandbox
	testcases []models.Testcase
	// makefile is the teacher makefile name, empty when the teacher sent none.
	makefile string
	cleanup  func()
}

// prepareTask puts the teacher files in the box first. Submission files never
// replace them, and when the teacher sent a makefile no student makefile enters
// the box, so the teacher recipe and drivers are the ones that run.
func prepareTask(submissionId, graderFilesURL, submissionFilesURL string, inputs, outputs []string) (preparedTask, error) {
	sbx, err := pool.Pool.Acquire()
	if err != nil {
		return preparedTask{}, fmt.Errorf("acquire sandbox: %w", err)
	}
	log.WithFields(log.Fields{"submissionId": submissionId, "sandboxId": sbx.BoxId}).Debug("Acquired sandbox")
	task := preparedTask{sbx: sbx, cleanup: func() { pool.Pool.Release(sbx) }}
	// Machinery does not give tasks a cancellable context. Download, host unzip,
	// sandbox unzip and compiler limits bound each operation independently.
	ctx := context.Background()
	taskDir := sandboxes.TaskDir(sbx.BoxId)
	task.testcases, err = prepareGraderFiles(ctx, sbx, taskDir, graderFilesURL, inputs, outputs)
	if err == nil {
		task.makefile = teacherMakefile(filepath.Join(taskDir, "files"))
		var studentMakefiles []string
		if task.makefile != "" {
			for _, name := range makefileNames {
				if name != task.makefile {
					studentMakefiles = append(studentMakefiles, name)
				}
			}
		}
		err = DownloadFileToSandbox(ctx, sbx, "submission.zip", submissionFilesURL, studentMakefiles...)
	}
	if err != nil {
		task.cleanup()
		return preparedTask{}, fmt.Errorf("prepare task: %w", err)
	}
	return task, nil
}

func teacherMakefile(graderDir string) string {
	for _, name := range makefileNames {
		if info, err := os.Lstat(filepath.Join(graderDir, name)); err == nil && info.Mode().IsRegular() {
			return name
		}
	}
	return ""
}
