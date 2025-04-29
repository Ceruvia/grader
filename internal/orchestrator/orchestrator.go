package orchestrator

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	language "github.com/Ceruvia/grader/internal/executor/languages"
	"github.com/Ceruvia/grader/internal/models"
)

const (
	TEMP_DIR = "tmp/"
)

func InitializeWorker(id int, jobs <-chan models.Submission, results chan<- []models.Verdict) {
	for sub := range jobs {
		fmt.Printf("Worker %d grading submission %s\n", id, sub.Id)

		workdir, err := os.MkdirTemp("", "example")
		if err != nil {
			panic(err)
		}
		defer os.Remove(workdir)

		log.Println(workdir)

		// TODO: change this dummy file download with real one
		err = copyAll("./tests/c_test/adt", workdir)
		if err != nil {
			panic(fmt.Sprintf("Failed to move files: %v", err))
		}

		exc, err := language.CreateNewCExecutor(
			os.DirFS("/"),
			strings.TrimPrefix(workdir, "/"),
			sub.BuildFiles,
			sub.TCInputFiles,
			sub.TCOutputFiles,
		)

		if err != nil {
			panic(err)
		}

		verdicts, _ := exc.Execute()

		results <- verdicts
	}
}

// TODO: remove this goofy ahh function man
func copyAll(srcDir, dstDir string) error {
	// Make sure destination exists
	err := os.MkdirAll(dstDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Walk through the source directory
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the root source directory itself
		if path == srcDir {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			// Create subdirectories
			return os.MkdirAll(destPath, info.Mode())
		} else {
			// Copy files (without deleting source)
			err = copyFile(path, destPath)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error while copying files: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	// Open the source file
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer in.Close()

	// Create the destination file
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer out.Close()

	// Copy contents
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Copy file permissions
	info, err := in.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}
	err = os.Chmod(dst, info.Mode())
	if err != nil {
		return fmt.Errorf("failed to set destination file permissions: %w", err)
	}

	return nil
}
