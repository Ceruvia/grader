package sandboxes_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/models"
	"github.com/Ceruvia/grader/internal/orchestrator/sandboxes"
)

func TestIsolate(t *testing.T) {
	t.Run("Creation", func(t *testing.T) {
		t.Run("Returns correct Isolate sandbox", func(t *testing.T) {
			sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 100)
			defer sbx.Cleanup()

			want := sandboxes.IsolateSandbox{
				IsolatePath:   "/usr/local/bin/isolate",
				BoxId:         100,
				AllowedDirs:   []string{},
				Filenames:     []string{},
				FileSizeLimit: 100 * 1024,
				MaxProcesses:  50,
				BoxDir:        "/var/local/lib/isolate/100/box",
			}

			tester.AssertNotError(t, err)
			tester.AssertDeep(t, *sbx, want)
		})

		t.Run("Returns error when configuration is false (boxId out of range)", func(t *testing.T) {
			_, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 100000000)
			tester.AssertIncludesError(t, err, "Sandbox ID out of range (allowed: 0-999)")
		})
	})

	t.Run("Setters", func(t *testing.T) {
		t.Run("SetTimeLimitInMiliseconds", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{}
			sbx.SetTimeLimitInMiliseconds(1000)

			want := sandboxes.IsolateSandbox{
				TimeLimit: 1000,
			}

			tester.AssertDeep(t, sbx, want)
		})

		t.Run("SetWallTimeLimitInMiliseconds", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{}
			sbx.SetWallTimeLimitInMiliseconds(1000)

			want := sandboxes.IsolateSandbox{
				WallTimeLimit: 1000,
			}

			tester.AssertDeep(t, sbx, want)
		})

		t.Run("SetMemoryLimitInKilobytes", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{}
			sbx.SetMemoryLimitInKilobytes(1024000)

			want := sandboxes.IsolateSandbox{
				MemoryLimit: 1024000,
			}

			tester.AssertDeep(t, sbx, want)
		})
	})

	t.Run("Getters", func(t *testing.T) {
		sbx := sandboxes.IsolateSandbox{
			BoxId:         100,
			BoxDir:        "/var/local/lib/isolate/100/box",
			TimeLimit:     1000,
			WallTimeLimit: 1000,
			MemoryLimit:   10240,
			FileSizeLimit: 1024000,
			MaxProcesses:  50,
			Filenames:     []string{"yo.c", "gurt.c"},
		}

		t.Run("GetBoxdir", func(t *testing.T) {
			got := sbx.GetBoxdir()
			want := sbx.BoxDir
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetBoxId", func(t *testing.T) {
			got := sbx.GetBoxId()
			want := sbx.BoxId
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetTimeLimit", func(t *testing.T) {
			got := sbx.GetTimeLimit()
			want := sbx.TimeLimit
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetWallTimeLimit", func(t *testing.T) {
			got := sbx.GetWallTimeLimit()
			want := sbx.WallTimeLimit
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetMemoryLimit", func(t *testing.T) {
			got := sbx.GetMemoryLimit()
			want := sbx.MemoryLimit
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetFileSizeLimit", func(t *testing.T) {
			got := sbx.GetFileSizeLimit()
			want := sbx.FileSizeLimit
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetMaxProcesses", func(t *testing.T) {
			got := sbx.GetMaxProcesses()
			want := sbx.MaxProcesses
			tester.AssertDeep(t, got, want)
		})

		t.Run("GetFilenamesInBox", func(t *testing.T) {
			got := sbx.GetFilenamesInBox()
			want := sbx.Filenames
			tester.AssertDeep(t, got, want)
		})

	})

	t.Run("AddFile", func(t *testing.T) {
		t.Run("Move and adds file into boxdir", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{
				Filenames: []string{},
				BoxDir:    "../../../tests/copy/dest",
			}

			err := sbx.AddFile("../../../tests/copy/source/file.c")
			defer os.Remove("../../../tests/copy/dest/file.c")

			tester.AssertNotError(t, err)

			if _, err := os.Stat("../../../tests/copy/dest/file.c"); err != nil {
				t.Errorf("file was not moved to Boxdir: %q", err)
			}

			got := sbx.Filenames
			want := []string{"file.c"}

			tester.AssertDeep(t, got, want)
		})

		t.Run("Returns error when file doesn't exist", func(t *testing.T) {
			sbx := sandboxes.IsolateSandbox{
				Filenames: []string{},
				BoxDir:    "tests/fake/destination",
			}

			err := sbx.AddFile("tests/fake/source/gaada.c")

			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
			}
		})
	})

	t.Run("ContainsFile", func(t *testing.T) {
		sbx := sandboxes.IsolateSandbox{
			Filenames: []string{"iexists.c"},
		}

		t.Run("Returns true when file exists in s.Filenames", func(t *testing.T) {
			got := sbx.ContainsFile("iexists.c")
			want := true

			tester.AssertDeep(t, got, want)
		})

		t.Run("Returns false when file doesn't exist in s.Filenames", func(t *testing.T) {
			got := sbx.ContainsFile("idontexists.c")
			want := false

			tester.AssertDeep(t, got, want)
		})
	})

	t.Run("GetFile", func(t *testing.T) {
		sbx := sandboxes.IsolateSandbox{
			BoxDir:    "../../../tests/copy/source",
			Filenames: []string{"file.c", "gurt.c"},
		}

		t.Run("Returns file contents", func(t *testing.T) {
			data, err := sbx.GetFile("file.c")

			tester.AssertNotError(t, err)

			got := string(data)
			want := "smth"

			tester.AssertDeep(t, got, want)
		})

		t.Run("Returns error when file is not in Boxdir", func(t *testing.T) {
			_, err := sbx.GetFile("gurt.c")
			tester.AssertError(t, err, os.ErrNotExist)
		})

		t.Run("Returns error when file is not in s.Filenames", func(t *testing.T) {
			_, err := sbx.GetFile("nada.c")
			tester.AssertCustomError(t, err, sandboxes.ErrFilenameNotInBox)
		})
	})

	t.Run("AddAllowedDirectory", func(t *testing.T) {
		sbx := sandboxes.IsolateSandbox{
			Filenames:   []string{},
			AllowedDirs: []string{},
		}

		t.Run("Adds directory to s.AllowedDirs", func(t *testing.T) {
			err := sbx.AddAllowedDirectory("/etc")
			tester.AssertNotError(t, err)
		})

		t.Run("Returns error when directory doesn't exist", func(t *testing.T) {
			err := sbx.AddAllowedDirectory("/apalahgaada")
			tester.AssertError(t, err, os.ErrNotExist)
		})
	})

	t.Run("BuildCommand", func(t *testing.T) {
		DummyRunCommand := command.GetCommandBuilder("gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello")

		BuildTests := []struct {
			Title            string
			Sandbox          sandboxes.IsolateSandbox
			RedirectionFiles sandboxes.RedirectionFiles
			ExpectedCommand  string
		}{
			{
				Title: "Basic",
				Sandbox: sandboxes.IsolateSandbox{
					IsolatePath:   "isolate",
					BoxId:         101,
					AllowedDirs:   []string{},
					Filenames:     []string{},
					FileSizeLimit: 100 * 1024,
					MaxProcesses:  50,
				},
				ExpectedCommand: "isolate -b 101 --dir=/etc -e --cg -p50 -f102400 --run -- gcc hello.c -o hello",
			},
			{
				Title: "Limits",
				Sandbox: sandboxes.IsolateSandbox{
					IsolatePath:   "isolate",
					BoxId:         102,
					AllowedDirs:   []string{},
					Filenames:     []string{},
					FileSizeLimit: 100 * 1024,
					MaxProcesses:  50,
					TimeLimit:     10000,
					WallTimeLimit: 10000,
					MemoryLimit:   10240,
				},
				ExpectedCommand: "isolate -b 102 --dir=/etc -e --cg -p50 -t10 -x0.5 -w10 --cg-mem=10240 -k10240 -f102400 --run -- gcc hello.c -o hello",
			},
			{
				Title: "Allowed Dir",
				Sandbox: sandboxes.IsolateSandbox{
					IsolatePath:   "isolate",
					BoxId:         103,
					AllowedDirs:   []string{"/usr/bin", "/var"},
					Filenames:     []string{},
					FileSizeLimit: 100 * 1024,
					MaxProcesses:  50,
				},
				ExpectedCommand: "isolate -b 103 --dir=/etc --dir=/usr/bin:rw --dir=/var:rw -e --cg -p50 -f102400 --run -- gcc hello.c -o hello",
			},
			{
				Title: "Redirections",
				Sandbox: sandboxes.IsolateSandbox{
					IsolatePath:   "isolate",
					BoxId:         104,
					AllowedDirs:   []string{},
					Filenames:     []string{},
					FileSizeLimit: 100 * 1024,
					MaxProcesses:  50,
				},
				RedirectionFiles: sandboxes.RedirectionFiles{
					StandardInputFilename:  "1.in",
					StandardOutputFilename: "1.out.expected",
					StandardErrorFilename:  "1.out.error",
					MetaFilename:           "1.out.meta",
				},
				ExpectedCommand: "isolate -b 104 --dir=/etc -e --cg -p50 -f102400 -i1.in -o1.out.expected -r1.out.error -M1.out.meta --run -- gcc hello.c -o hello",
			},
			{
				Title: "All",
				Sandbox: sandboxes.IsolateSandbox{
					IsolatePath:   "isolate",
					BoxId:         105,
					AllowedDirs:   []string{"/usr/bin", "/var"},
					Filenames:     []string{},
					FileSizeLimit: 100 * 1024,
					MaxProcesses:  50,
					TimeLimit:     10000,
					WallTimeLimit: 10000,
					MemoryLimit:   10240,
				},
				RedirectionFiles: sandboxes.RedirectionFiles{
					StandardInputFilename:  "1.in",
					StandardOutputFilename: "1.out.expected",
					StandardErrorFilename:  "1.out.error",
					MetaFilename:           "1.out.meta",
				},
				ExpectedCommand: "isolate -b 105 --dir=/etc --dir=/usr/bin:rw --dir=/var:rw -e --cg -p50 -t10 -x0.5 -w10 --cg-mem=10240 -k10240 -f102400 -i1.in -o1.out.expected -r1.out.error -M1.out.meta --run -- gcc hello.c -o hello",
			},
		}

		for _, test := range BuildTests {
			t.Run(fmt.Sprintf("Returns expected command when using %s configuration", test.Title), func(t *testing.T) {
				got := test.Sandbox.BuildCommand(*DummyRunCommand, test.RedirectionFiles)
				if got.BuildFullCommand() != test.ExpectedCommand {
					t.Errorf("got %q, expected %q", got.BuildFullCommand(), test.ExpectedCommand)
				}
			})
		}
	})

	t.Run("Execute", func(t *testing.T) {
		emptyCommand := *command.GetCommandBuilder("true")

		checkEmptyCommand := func(cmd command.CommandBuilder) bool {
			return cmd.Program == "true"
		}

		E2ETests := []struct {
			Title                 string
			CodeToCompileFilepath string
			RunCommand            command.CommandBuilder
			SecondRunCommand      command.CommandBuilder
			ExpectedStatus        models.SandboxExecutionStatus
			ExpectedMessage       string
		}{
			{
				Title:                 "Compile hello.c and run",
				CodeToCompileFilepath: "../../../tests/c_test/hello.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
				SecondRunCommand:      *command.GetCommandBuilder("./hello"),
				ExpectedStatus:        models.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile hello.c",
				CodeToCompileFilepath: "../../../tests/c_test/hello.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile empty.c",
				CodeToCompileFilepath: "../../../tests/c_test/uncompileable/empty.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("empty.c").AddArgs("-o").AddArgs("empty"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile infiniterecursion.c and run",
				CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/infiniterecursion.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniterecursion.c").AddArgs("-o").AddArgs("infinite"),
				SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
				ExpectedStatus:        models.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 9",
			},
			{
				Title:                 "Compile noinclude.c",
				CodeToCompileFilepath: "../../../tests/c_test/uncompileable/noinclude.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("noinclude.c").AddArgs("-o").AddArgs("noinclude"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile nullpointer.c and run",
				CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/nullpointer.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("nullpointer.c").AddArgs("-o").AddArgs("nullpointer"),
				SecondRunCommand:      *command.GetCommandBuilder("./nullpointer"),
				ExpectedStatus:        models.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 11",
			},
			{
				Title:                 "Compile outofbounds.c and run",
				CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/outofbounds.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("outofbounds.c").AddArgs("-o").AddArgs("outofbounds"),
				SecondRunCommand:      *command.GetCommandBuilder("./outofbounds"),
				ExpectedStatus:        models.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 6",
			},
			{
				Title:                 "Compile syntaxerror.c",
				CodeToCompileFilepath: "../../../tests/c_test/uncompileable/syntaxerror.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("syntaxerror.c").AddArgs("-o").AddArgs("syntaxerror"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile typemismatch.c",
				CodeToCompileFilepath: "../../../tests/c_test/uncompileable/typemismatch.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("typemismatch.c").AddArgs("-o").AddArgs("typemismatch"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile unfoundfunc.c",
				CodeToCompileFilepath: "../../../tests/c_test/uncompileable/unfoundfunc.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("unfoundfunc.c").AddArgs("-o").AddArgs("unfoundfunc"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        models.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile infiniteloop.c and run",
				CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/infiniteloop.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniteloop.c").AddArgs("-o").AddArgs("infinite"),
				SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
				ExpectedStatus:        models.TIMED_OUT,
				ExpectedMessage:       "Time limit exceeded (wall clock)",
			},
		}

		for boxnum, test := range E2ETests {
			t.Run(fmt.Sprintf("Returns expected status and message when running %q", test.Title), func(t *testing.T) {
				sbx, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", boxnum+110)
				defer sbx.Cleanup()

				if err != nil {
					t.Fatal(err)
				}

				err = sbx.AddFile(test.CodeToCompileFilepath)
				if err != nil {
					t.Fatal(err)
				}

				red := sandboxes.CreateRedirectionFiles(sbx.BoxDir)
				err = red.CreateNewMetaFileAndRedirect("comp.meta")
				if err != nil {
					t.Fatal(err)
				}

				err = red.CreateNewStandardOutputFileAndRedirect("comp.out")
				if err != nil {
					t.Fatal(err)
				}

				err = red.RedirectStandardError("comp.out")
				if err != nil {
					t.Fatal(err)
				}

				sbx.SetTimeLimitInMiliseconds(1000)
				sbx.SetWallTimeLimitInMiliseconds(1000)
				sbx.SetMemoryLimitInKilobytes(10240)
				sbx.AddAllowedDirectory("/etc")

				res, err := sbx.Execute(test.RunCommand, red)

				if err != nil {
					t.Fatal(err)
				}

				if !checkEmptyCommand(test.SecondRunCommand) {
					secondRed := sandboxes.CreateRedirectionFiles(sbx.BoxDir)
					err = secondRed.CreateNewMetaFileAndRedirect("run.meta")
					if err != nil {
						t.Fatal(err)
					}

					err = secondRed.CreateNewStandardOutputFileAndRedirect("run.out")
					if err != nil {
						t.Fatal(err)
					}

					err = secondRed.RedirectStandardError("run.out")
					if err != nil {
						t.Fatal(err)
					}

					res, err = sbx.Execute(test.SecondRunCommand, secondRed)
					if err != nil {
						t.Fatal(err)
					}
				}

				if res.Status != test.ExpectedStatus || res.Message != test.ExpectedMessage {
					t.Errorf("expected status %s and message %s, instead got %+v", test.ExpectedStatus, test.ExpectedMessage, res)
				}
			})
		}
	})
}

func TestRedirections(t *testing.T) {
	t.Run("Creation And Redirection", func(t *testing.T) {
		t.Run("Create and redirects meta file", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/sandbox",
			}
			err := red.CreateNewMetaFileAndRedirect("_isolate.meta")
			defer deleteFile("../../../tests/sandbox/_isolate.meta")

			tester.AssertNotError(t, err)

			if _, err := os.Stat("../../../tests/sandbox/_isolate.meta"); err != nil {
				t.Fatalf("file was not created: %q", err.Error())
			}

			want := sandboxes.RedirectionFiles{
				Boxdir:       "../../../tests/sandbox",
				MetaFilename: "../../../tests/sandbox/_isolate.meta",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("Create and redirects standard input file", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/sandbox",
			}
			err := red.CreateNewStandardInputFileAndRedirect("input.in")
			defer deleteFile("../../../tests/sandbox/input.in")

			tester.AssertNotError(t, err)

			if _, err := os.Stat("../../../tests/sandbox/input.in"); err != nil {
				t.Fatalf("file was not created: %q", err.Error())
			}

			want := sandboxes.RedirectionFiles{
				Boxdir:                "../../../tests/sandbox",
				StandardInputFilename: "input.in",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("Create and redirects standard output file", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/sandbox",
			}
			err := red.CreateNewStandardOutputFileAndRedirect("output.out")
			defer deleteFile("../../../tests/sandbox/output.out")

			tester.AssertNotError(t, err)

			if _, err := os.Stat("../../../tests/sandbox/output.out"); err != nil {
				t.Fatalf("file was not created: %q", err.Error())
			}

			want := sandboxes.RedirectionFiles{
				Boxdir:                 "../../../tests/sandbox",
				StandardOutputFilename: "output.out",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("Create and redirects standard error file", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/sandbox",
			}
			err := red.CreateNewStandardErrorFileAndRedirect("error.err")
			defer deleteFile("../../../tests/sandbox/error.err")

			tester.AssertNotError(t, err)

			if _, err := os.Stat("../../../tests/sandbox/error.err"); err != nil {
				t.Fatalf("file was not created: %q", err.Error())
			}

			want := sandboxes.RedirectionFiles{
				Boxdir:                "../../../tests/sandbox",
				StandardErrorFilename: "error.err",
			}

			tester.AssertDeep(t, red, want)
		})
	})

	t.Run("Setters", func(t *testing.T) {
		t.Run("RedirectMeta", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/copy/source",
			}
			err := red.RedirectMeta("file.c")

			tester.AssertNotError(t, err)

			want := sandboxes.RedirectionFiles{
				Boxdir:       "../../../tests/copy/source",
				MetaFilename: "../../../tests/copy/source/file.c",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("RedirectStandardInput", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/copy/source",
			}
			err := red.RedirectStandardInput("file.c")

			tester.AssertNotError(t, err)

			want := sandboxes.RedirectionFiles{
				Boxdir:                "../../../tests/copy/source",
				StandardInputFilename: "file.c",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("RedirectStandardOutput", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/copy/source",
			}
			err := red.RedirectStandardOutput("file.c")

			tester.AssertNotError(t, err)

			want := sandboxes.RedirectionFiles{
				Boxdir:                 "../../../tests/copy/source",
				StandardOutputFilename: "file.c",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("RedirectStandardMeta", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				Boxdir: "../../../tests/copy/source",
			}
			err := red.RedirectStandardError("file.c")

			tester.AssertNotError(t, err)

			want := sandboxes.RedirectionFiles{
				Boxdir:                "../../../tests/copy/source",
				StandardErrorFilename: "file.c",
			}

			tester.AssertDeep(t, red, want)
		})

		t.Run("ResetRedirection", func(t *testing.T) {
			red := sandboxes.RedirectionFiles{
				StandardInputFilename:  "1.in",
				StandardOutputFilename: "1.out.expected",
				StandardErrorFilename:  "1.err",
			}

			red.ResetRedirection()

			want := sandboxes.RedirectionFiles{}

			tester.AssertDeep(t, red, want)
		})
	})
}

func TestParser(t *testing.T) {
	Tests := []struct {
		Title, Filename string
		Expected        models.SandboxExecutionResult
	}{
		{"Success", "success.meta", models.SandboxExecutionResult{
			Status:     models.ZERO_EXIT_CODE,
			ExitSignal: 0,
			ExitCode:   0,
			Time:       31,
			WallTime:   31,
			Memory:     7900,
			Message:    "",
			IsKilled:   false,
		}},
		{"Runtime Error", "re.meta", models.SandboxExecutionResult{
			Status:     models.NONZERO_EXIT_CODE,
			ExitSignal: 0,
			ExitCode:   1,
			Time:       25,
			WallTime:   50,
			Memory:     7220,
			Message:    "Exited with error status 1",
			IsKilled:   false,
		}},
		{"Killed on Signal", "sigkill.meta", models.SandboxExecutionResult{
			Status:     models.KILLED_ON_SIGNAL,
			ExitSignal: 9,
			ExitCode:   0,
			Time:       10,
			WallTime:   32,
			Memory:     10240,
			Message:    "Caught fatal signal 9",
			IsKilled:   false,
		}},
		{"Time Limit Exceeded", "tle.meta", models.SandboxExecutionResult{
			Status:     models.TIMED_OUT,
			ExitSignal: 0,
			ExitCode:   0,
			Time:       1077,
			WallTime:   1100,
			Memory:     7956,
			Message:    "Time limit exceeded (wall clock)",
			IsKilled:   true,
		}},
	}

	for _, test := range Tests {
		t.Run(fmt.Sprintf("Parse %+v", test.Title), func(t *testing.T) {
			got, err := sandboxes.ParseMetaResult("../../../tests/sandbox/parse/" + test.Filename)
			tester.AssertNotError(t, err)
			tester.AssertDeep(t, got, test.Expected)
		})
	}

}

func deleteFile(filepath string) error {
	return os.Remove(filepath)
}
