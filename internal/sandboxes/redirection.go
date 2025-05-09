package sandboxes

import "os"

type RedirectionFiles struct {
	StandardInputFilename  string
	StandardOutputFilename string
	StandardErrorFilename  string
	MetaFilename           string
}

func CreateRedirectionFiles() RedirectionFiles {
	return RedirectionFiles{}
}

func (r *RedirectionFiles) ResetRedirection() {
	r.StandardInputFilename = ""
	r.StandardOutputFilename = ""
	r.StandardErrorFilename = ""
	r.MetaFilename = ""
}

func (r *RedirectionFiles) RedirectMeta(boxdir, filenameInsideBox string) error {
	if _, err := os.Stat(boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.MetaFilename = boxdir + "/" + filenameInsideBox
	return nil
}

func (r *RedirectionFiles) RedirectStandardInput(boxdir, filenameInsideBox string) error {
	if _, err := os.Stat(boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardInputFilename = boxdir + "/" + filenameInsideBox
	return nil
}

func (r *RedirectionFiles) RedirectStandardOutput(boxdir, filenameInsideBox string) error {
	if _, err := os.Stat(boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardOutputFilename = boxdir + "/" + filenameInsideBox
	return nil
}

func (r *RedirectionFiles) RedirectStandardError(boxdir, filenameInsideBox string) error {
	if _, err := os.Stat(boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.StandardErrorFilename = boxdir + "/" + filenameInsideBox
	return nil
}
