package compilers

import (
	"strings"

	"github.com/Ceruvia/grader/internal/languages"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/sandboxes"
)

// CompileSubmission uses the same source selection, builder path and limits for
// grading and compile-only checks. A positive message limit bounds diagnostics.
func CompileSubmission(sandbox sandboxes.Sandbox, submission models.Submission, messageLimitBytes int64) (CompilerResult, error) {
	if submission.GetLanguage() == nil {
		return CompilerResult{}, languages.ErrLanguageNotExist
	}
	compiler, err := PrepareSourceFileCompiler(sandbox, submission.GetCompileLanguage())
	if err != nil {
		return CompilerResult{}, err
	}
	sourceFilenames := []string{}
	if !submission.IsBuilder() {
		for _, filename := range sandbox.GetFilenamesInBox() {
			for _, extension := range submission.GetLanguage().GetAllowedExtention() {
				if strings.HasSuffix(filename, "."+extension) {
					sourceFilenames = append(sourceFilenames, filename)
				}
			}
		}
	}
	return compiler.Compile(submission.GetCompileFilenameOrScript(), sourceFilenames, messageLimitBytes), nil
}
