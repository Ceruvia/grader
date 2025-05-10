package sandboxes

import "os"

type RedirectionFiles struct {
	Boxdir                 string
	StandardInputFilename  string
	StandardOutputFilename string
	StandardErrorFilename  string
	MetaFilename           string
}

func CreateRedirectionFiles(boxdir string) RedirectionFiles {
	return RedirectionFiles{
		Boxdir: boxdir,
	}
}

func (r *RedirectionFiles) ResetRedirection() {
	r.StandardInputFilename = ""
	r.StandardOutputFilename = ""
	r.StandardErrorFilename = ""
	r.MetaFilename = ""
}

func (r *RedirectionFiles) RedirectMeta(filenameInsideBox string) error {
	if _, err := os.Stat(r.Boxdir + "/" + filenameInsideBox); err != nil {
		return err
	}
	r.MetaFilename = r.Boxdir + "/" + filenameInsideBox
	return nil
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
