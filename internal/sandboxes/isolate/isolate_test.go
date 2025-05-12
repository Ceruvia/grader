package isolate_test

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Ceruvia/grader/internal/command"
	"github.com/Ceruvia/grader/internal/sandboxes"
	"github.com/Ceruvia/grader/internal/sandboxes/isolate"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestCreateIsolateSandbox(t *testing.T) {
	t.Run("it should succesfully create an Isolate sandbox", func(t *testing.T) {
		sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", 990)
		defer sbx.Cleanup()

		want := isolate.IsolateSandbox{
			IsolatePath:   "/usr/local/bin/isolate",
			BoxId:         990,
			AllowedDirs:   []string{},
			Filenames:     []string{},
			FileSizeLimit: 100 * 1024,
			MaxProcesses:  50,
			BoxDir:        "/var/local/lib/isolate/990/box",
		}

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		utils.AssertDeep(t, sbx, want)
	})
}

func TestAddFile(t *testing.T) {
	t.Run("it should add the file to sbx.Filenames", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "../../../tests/copy/dest",
		}

		err := sbx.AddFile("../../../tests/copy/source/file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		got := sbx.Filenames
		want := []string{"file.c"}

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should copy the file to sbx.Boxdir", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "../../../tests/copy/dest",
		}

		err := sbx.AddFile("../../../tests/copy/source/file.c")
		defer os.Remove("../../../tests/copy/dest/file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		if _, err := os.Stat("../../../tests/copy/dest/file.c"); err != nil {
			t.Errorf("file was not moved to Boxdir: %q", err)
		}
	})

	t.Run("it should return error when file doesn't exist", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{
			Filenames: []string{},
			BoxDir:    "tests/fake/destination",
		}

		err := sbx.AddFile("tests/fake/source/gaada.c")

		if err == nil {
			t.Fatalf("didn't get an error when expecting: %q", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
		}
	})
}

func TestContainsFile(t *testing.T) {
	sbx := isolate.IsolateSandbox{
		Filenames: []string{"iexists.c"},
	}
	t.Run("it should return True when file is in sbx.Filenames", func(t *testing.T) {
		got := sbx.ContainsFile("iexists.c")
		want := true

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should return False when file doesn't exists in sbx.Filenames", func(t *testing.T) {
		got := sbx.ContainsFile("idontexists.c")
		want := false

		utils.AssertDeep(t, got, want)
	})
}

func TestGetFile(t *testing.T) {
	sbx := isolate.IsolateSandbox{
		BoxDir:    "../../../tests/copy/source",
		Filenames: []string{"file.c"},
	}

	t.Run("it should be able to get a file", func(t *testing.T) {
		data, err := sbx.GetFile("file.c")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}

		got := string(data)
		want := "smth"

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should return error when not in sbx.Filenames", func(t *testing.T) {
		_, err := sbx.GetFile("nada.c")

		if err == nil {
			t.Fatalf("didn't get an error when expecting: %q", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
		}
	})
}

func TestAddAllowedDirectory(t *testing.T) {
	sbx := isolate.IsolateSandbox{
		Filenames:   []string{},
		AllowedDirs: []string{},
	}

	t.Run("it should add an existing directory", func(t *testing.T) {
		err := sbx.AddAllowedDirectory("/etc")

		if err != nil {
			t.Fatalf("got an error when expecting none: %q", err)
		}
	})

	t.Run("it should error when directory doesn't exist", func(t *testing.T) {
		err := sbx.AddAllowedDirectory("/apalahgaada")

		if err == nil {
			t.Fatalf("didn't get an error when expecting: %q", err)
		}

		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf(`should've gotten "no such file or directory", instead got %q`, err)
		}
	})
}

func TestSetters(t *testing.T) {
	t.Run("it should be able to set time limit", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{}
		sbx.SetTimeLimitInMiliseconds(1000)

		want := isolate.IsolateSandbox{
			TimeLimit: 1000,
		}

		utils.AssertDeep(t, sbx, want)
	})

	t.Run("it should be able to set wall time limit", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{}
		sbx.SetWallTimeLimitInMiliseconds(1000)

		want := isolate.IsolateSandbox{
			WallTimeLimit: 1000,
		}

		utils.AssertDeep(t, sbx, want)
	})

	t.Run("it should be able to set memory limit", func(t *testing.T) {
		sbx := isolate.IsolateSandbox{}
		sbx.SetMemoryLimitInKilobytes(1024000)

		want := isolate.IsolateSandbox{
			MemoryLimit: 1024000,
		}

		utils.AssertDeep(t, sbx, want)
	})
}

func TestBuildCommand(t *testing.T) {
	DummyRunCommand := command.GetCommandBuilder("gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello")

	BuildTests := []struct {
		Title            string
		Sandbox          isolate.IsolateSandbox
		RedirectionFiles sandboxes.RedirectionFiles
		ExpectedCommand  string
	}{
		{
			Title: "Basic",
			Sandbox: isolate.IsolateSandbox{
				IsolatePath:   "isolate",
				BoxId:         990,
				AllowedDirs:   []string{},
				Filenames:     []string{},
				FileSizeLimit: 100 * 1024,
				MaxProcesses:  50,
			},
			ExpectedCommand: "isolate -b 990 -e --cg -p50 -f102400 --run -- gcc hello.c -o hello",
		},
		{
			Title: "Limits",
			Sandbox: isolate.IsolateSandbox{
				IsolatePath:   "isolate",
				BoxId:         990,
				AllowedDirs:   []string{},
				Filenames:     []string{},
				FileSizeLimit: 100 * 1024,
				MaxProcesses:  50,
				TimeLimit:     10000,
				WallTimeLimit: 10000,
				MemoryLimit:   10240,
			},
			ExpectedCommand: "isolate -b 990 -e --cg -p50 -t10 -x0.5 -w10 --cg-mem=10240 -k10240 -f102400 --run -- gcc hello.c -o hello",
		},
		{
			Title: "Allowed Dir",
			Sandbox: isolate.IsolateSandbox{
				IsolatePath:   "isolate",
				BoxId:         990,
				AllowedDirs:   []string{"/usr/bin", "/var"},
				Filenames:     []string{},
				FileSizeLimit: 100 * 1024,
				MaxProcesses:  50,
			},
			ExpectedCommand: "isolate -b 990 --dir=/usr/bin:rw --dir=/var:rw -e --cg -p50 -f102400 --run -- gcc hello.c -o hello",
		},
		{
			Title: "Redirections",
			Sandbox: isolate.IsolateSandbox{
				IsolatePath:   "isolate",
				BoxId:         990,
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
			ExpectedCommand: "isolate -b 990 -e --cg -p50 -f102400 -i1.in -o1.out.expected -r1.out.error -M1.out.meta --run -- gcc hello.c -o hello",
		},
		{
			Title: "All",
			Sandbox: isolate.IsolateSandbox{
				IsolatePath:   "isolate",
				BoxId:         990,
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
			ExpectedCommand: "isolate -b 990 --dir=/usr/bin:rw --dir=/var:rw -e --cg -p50 -t10 -x0.5 -w10 --cg-mem=10240 -k10240 -f102400 -i1.in -o1.out.expected -r1.out.error -M1.out.meta --run -- gcc hello.c -o hello",
		},
	}

	for _, test := range BuildTests {
		t.Run(fmt.Sprintf("it should be able to create build command for sandbox with %s configuration", test.Title), func(t *testing.T) {
			got := test.Sandbox.BuildCommand(*DummyRunCommand, test.RedirectionFiles)
			if got.BuildFullCommand() != test.ExpectedCommand {
				t.Errorf("got %q, expected %q", got.BuildFullCommand(), test.ExpectedCommand)
			}
		})
	}
}

func TestEndToEnd(t *testing.T) {
	emptyCommand := *command.GetCommandBuilder("true")

	checkEmptyCommand := func(cmd command.CommandBuilder) bool {
		return cmd.Program == "true"
	}

	E2ETests := []struct {
		Title                 string
		CodeToCompileFilepath string
		RunCommand            command.CommandBuilder
		SecondRunCommand      command.CommandBuilder
		ExpectedStatus        sandboxes.SandboxExecutionStatus
		ExpectedMessage       string
	}{
		{
			Title:                 "Compile hello.c and run",
			CodeToCompileFilepath: "../../../tests/c_test/hello.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
			SecondRunCommand:      *command.GetCommandBuilder("./hello"),
			ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
			ExpectedMessage:       "",
		},
		{
			Title:                 "Compile hello.c",
			CodeToCompileFilepath: "../../../tests/c_test/hello.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
			ExpectedMessage:       "",
		},
		{
			Title:                 "Compile empty.c",
			CodeToCompileFilepath: "../../../tests/c_test/uncompileable/empty.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("empty.c").AddArgs("-o").AddArgs("empty"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
			ExpectedMessage:       "Exited with error status 1",
		},
		{
			Title:                 "Compile infiniterecursion.c and run",
			CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/infiniterecursion.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniterecursion.c").AddArgs("-o").AddArgs("infinite"),
			SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
			ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
			ExpectedMessage:       "Caught fatal signal 9",
		},
		{
			Title:                 "Compile noinclude.c",
			CodeToCompileFilepath: "../../../tests/c_test/uncompileable/noinclude.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("noinclude.c").AddArgs("-o").AddArgs("noinclude"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
			ExpectedMessage:       "Exited with error status 1",
		},
		{
			Title:                 "Compile nullpointer.c and run",
			CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/nullpointer.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("nullpointer.c").AddArgs("-o").AddArgs("nullpointer"),
			SecondRunCommand:      *command.GetCommandBuilder("./nullpointer"),
			ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
			ExpectedMessage:       "Caught fatal signal 11",
		},
		{
			Title:                 "Compile outofbounds.c and run",
			CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/outofbounds.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("outofbounds.c").AddArgs("-o").AddArgs("outofbounds"),
			SecondRunCommand:      *command.GetCommandBuilder("./outofbounds"),
			ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
			ExpectedMessage:       "Caught fatal signal 6",
		},
		{
			Title:                 "Compile syntaxerror.c",
			CodeToCompileFilepath: "../../../tests/c_test/uncompileable/syntaxerror.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("syntaxerror.c").AddArgs("-o").AddArgs("syntaxerror"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
			ExpectedMessage:       "Exited with error status 1",
		},
		{
			Title:                 "Compile typemismatch.c",
			CodeToCompileFilepath: "../../../tests/c_test/uncompileable/typemismatch.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("typemismatch.c").AddArgs("-o").AddArgs("typemismatch"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
			ExpectedMessage:       "Exited with error status 1",
		},
		{
			Title:                 "Compile unfoundfunc.c",
			CodeToCompileFilepath: "../../../tests/c_test/uncompileable/unfoundfunc.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("unfoundfunc.c").AddArgs("-o").AddArgs("unfoundfunc"),
			SecondRunCommand:      emptyCommand,
			ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
			ExpectedMessage:       "Exited with error status 1",
		},
		{
			Title:                 "Compile infiniteloop.c and run",
			CodeToCompileFilepath: "../../../tests/c_test/runtimeerror/infiniteloop.c",
			RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniteloop.c").AddArgs("-o").AddArgs("infinite"),
			SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
			ExpectedStatus:        sandboxes.TIMED_OUT,
			ExpectedMessage:       "Time limit exceeded (wall clock)",
		},
	}

	for boxnum, test := range E2ETests {
		t.Run(fmt.Sprintf("it should be able to %q with expected results", test.Title), func(t *testing.T) {
			sbx, err := isolate.CreateIsolateSandbox("/usr/local/bin/isolate", boxnum)
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
}
