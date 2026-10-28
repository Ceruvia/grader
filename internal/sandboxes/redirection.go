package sandboxes

import (
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// TaskRoot holds one host-only directory per box. The box can never reach it.
const TaskRoot = "/var/lib/grader/tasks"

// TaskDir is the host-only directory of a box: grader files and isolate meta files.
func TaskDir(boxId int) string {
	return filepath.Join(TaskRoot, strconv.Itoa(boxId))
}

// MetaDir holds the isolate meta files of a box. Isolate runs as root and writes
// the meta file outside the sandbox, so it must never be a path the program controls.
func MetaDir(boxId int) string {
	return filepath.Join(TaskDir(boxId), "meta")
}

type RedirectionFiles struct {
	Boxdir                 string
	MetaDir                string
	StandardInputFilename  string
	StandardOutputFilename string
	StandardErrorFilename  string
	MetaFilename           string
}

func CreateRedirectionFiles(sbx Sandbox) RedirectionFiles {
	return RedirectionFiles{
		Boxdir:  sbx.GetBoxdir(),
		MetaDir: MetaDir(sbx.GetBoxId()),
	}
}

func (r *RedirectionFiles) ResetRedirection() {
	r.StandardInputFilename = ""
	r.StandardOutputFilename = ""
	r.StandardErrorFilename = ""
	r.MetaFilename = ""
}

func (r *RedirectionFiles) RedirectStandardInput(filenameInsideBox string) error {
	if _, err := os.Stat(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardInputFilename = filenameInsideBox
	return nil
}

func (r *RedirectionFiles) RedirectStandardOutput(filenameInsideBox string) error {
	if _, err := os.Stat(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardOutputFilename = filenameInsideBox
	return nil
}

func (r *RedirectionFiles) RedirectStandardError(filenameInsideBox string) error {
	if _, err := os.Stat(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardErrorFilename = filenameInsideBox
	return nil
}

// createFresh replaces whatever sits at path (a symlink or FIFO planted by a
// submission included) with a new empty regular file. It never follows a link.
func createFresh(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0666)
	if err != nil {
		return err
	}
	return f.Close()
}

// CreateNewMetaFileAndRedirect creates the meta file in the host-only MetaDir.
func (r *RedirectionFiles) CreateNewMetaFileAndRedirect(filename string) error {
	if err := os.MkdirAll(r.MetaDir, 0700); err != nil {
		return err
	}
	path := filepath.Join(r.MetaDir, filepath.Base(filename))
	if err := createFresh(path); err != nil {
		return err
	}
	r.MetaFilename = path
	return nil
}

func (r *RedirectionFiles) CreateNewStandardInputFileAndRedirect(filenameInsideBox string) error {
	if err := createFresh(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardInputFilename = filenameInsideBox
	return nil
}

func (r *RedirectionFiles) CreateNewStandardOutputFileAndRedirect(filenameInsideBox string) error {
	if err := createFresh(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardOutputFilename = filenameInsideBox
	return nil
}

func (r *RedirectionFiles) CreateNewStandardErrorFileAndRedirect(filenameInsideBox string) error {
	if err := createFresh(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardErrorFilename = filenameInsideBox
	return nil
}
