package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/evaluator"
	"github.com/Ceruvia/grader/internal/pool"
)

func TestTeacherFilesWin(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)

	// If a student recipe writes pwned, the program returns the wrong output.
	const cSource = "#include <stdio.h>\nint main(void) { FILE *f = fopen(\"pwned\", \"r\"); if (f) { puts(\"pwned\"); fclose(f); } else { puts(\"6\"); } return 0; }\n"
	const teacherRecipe = "teacher:\n\tgcc -o main main.c\n"
	const studentRecipe = "teacher:\n\t@echo pwned > pwned\n\tgcc -o main main.c\n"
	for _, tc := range []struct {
		name                string
		forbiddenAlias      string
		teacher, submission map[string]string
	}{
		{"same Makefile name", "", map[string]string{"Makefile": teacherRecipe}, map[string]string{"Makefile": studentRecipe}},
		{"GNUmakefile shadows lowercase makefile", "GNUmakefile", map[string]string{"makefile": teacherRecipe}, map[string]string{"GNUmakefile": studentRecipe}},
		{"alternate student Makefile removed", "Makefile", map[string]string{"makefile": teacherRecipe}, map[string]string{"Makefile": studentRecipe}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.teacher["1.in"], tc.teacher["1.out"] = "", "6\n"
			tc.submission["main.c"] = cSource
			if tc.forbiddenAlias != "" {
				probe := fmt.Sprintf("int main(void) { FILE *alias = fopen(%q, \"r\"); if (alias) { puts(\"pwned\"); fclose(alias); return 0; }", tc.forbiddenAlias)
				tc.submission["main.c"] = strings.Replace(cSource, "int main(void) {", probe, 1)
			}
			server := archiveServer(t, map[string]map[string]string{"/grader.zip": tc.teacher, "/submission.zip": tc.submission})
			result, err := GradeBlackboxWithBuilder("teacher-makefile-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "teacher", "main")
			if err != nil {
				t.Fatal(err)
			}
			var got evaluator.GradingResult
			if err := json.Unmarshal([]byte(result), &got); err != nil {
				t.Fatal(err)
			}
			if !got.IsSuccess || len(got.TestcaseGradingResult) != 1 || got.TestcaseGradingResult[0].Verdict != models.VerdictAC {
				t.Fatalf("teacher recipe must run without the pwned side effect, got %s", result)
			}
		})
	}
}

func TestStudentMakefileWhenTeacherHasNone(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {"1.in": "", "1.out": "6\n"},
		"/submission.zip": {
			"Makefile": "student:\n\t@echo built > student-build\n\tgcc -o main main.c\n",
			"main.c":   "#include <stdio.h>\nint main(void) { FILE *f = fopen(\"student-build\", \"r\"); if (!f) return 1; fclose(f); puts(\"6\"); return 0; }\n",
		},
	})
	result, err := GradeBlackboxWithBuilder("student-makefile-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "student", "main")
	if err != nil {
		t.Fatal(err)
	}
	var got evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsSuccess || len(got.TestcaseGradingResult) != 1 || got.TestcaseGradingResult[0].Verdict != models.VerdictAC {
		t.Fatalf("student recipe must be used when teacher has no makefile, got %s", result)
	}
}

func TestTeacherHaskellDriverCannotBeReplaced(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {
			"1.in": "", "1.out": "42\n",
			"makefile": "haskell:\n\tghc --make Main\n",
			"Main.hs":  "import qualified Student\nmain = print Student.answer\n",
		},
		"/submission.zip": {
			"Main.hs":    "main = print (42 :: Int)\n",
			"Student.hs": "module Student where\nanswer :: Int\nanswer = 0\n",
		},
	})
	result, err := GradeBlackboxWithBuilder("teacher-driver-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "Haskell", "Makefile", "haskell", "Main")
	if err != nil {
		t.Fatal(err)
	}
	var got evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsSuccess || len(got.TestcaseGradingResult) != 1 || got.TestcaseGradingResult[0].Verdict != models.VerdictWA {
		t.Fatalf("wrong student module must get WA through the teacher driver, got %s", result)
	}
}

func TestTeacherMakefileIgnoresPrebuiltStudentTarget(t *testing.T) {
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", 1); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Pool.Close)
	server := archiveServer(t, map[string]map[string]string{
		"/grader.zip": {
			"1.in": "", "1.out": "6\n",
			"makefile": "main: driver.c student.c\n\tgcc -o main driver.c student.c\n",
			"driver.c": "#include <stdio.h>\nint answer(void);\nint main(void) { printf(\"%d\\n\", answer()); return 0; }\n",
		},
		"/submission.zip": {
			"student.c": "int answer(void) { return 0; }\n",
			// Newer than every prerequisite, so make without -B would call it up to date.
			"main": prebuiltEntry + "#!/bin/sh\necho 6\n",
		},
	})
	result, err := GradeBlackboxWithBuilder("prebuilt-target-test", server.URL+"/grader.zip", server.URL+"/submission.zip", []string{"1.in"}, []string{"1.out"}, 1000, 65536, "C", "Makefile", "main", "main")
	if err != nil {
		t.Fatal(err)
	}
	var got evaluator.GradingResult
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatal(err)
	}
	if !got.IsSuccess || len(got.TestcaseGradingResult) != 1 || got.TestcaseGradingResult[0].Verdict != models.VerdictWA {
		t.Fatalf("teacher recipe must rebuild main from the wrong student module, got %s", result)
	}
}
