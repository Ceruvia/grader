//go:build linux && integration

package sandboxes_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Ceruvia/grader/internal/helper/command"
	"github.com/Ceruvia/grader/internal/helper/tester"
	"github.com/Ceruvia/grader/internal/sandboxes"
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

	t.Run("AddAllowedDirectory", func(t *testing.T) {
		t.Run("Program in the box reads the directory only after it is added", func(t *testing.T) {
			// /var/lib is not in isolate's default mounts, so only AddAllowedDirectory can expose it.
			dir, err := os.MkdirTemp("/var/lib", "allowed-dir-")
			tester.AssertNotError(t, err)
			t.Cleanup(func() { _ = os.RemoveAll(dir) })
			tester.AssertNotError(t, os.Chmod(dir, 0755))
			tester.AssertNotError(t, os.WriteFile(filepath.Join(dir, "data.txt"), []byte("allowed-dir-canary\n"), 0644))

			box, err := sandboxes.CreateIsolateSandbox("/usr/local/bin/isolate", 504)
			tester.AssertNotError(t, err)
			t.Cleanup(func() { _ = box.Cleanup() })
			box.SetWallTimeLimitInMiliseconds(5000)
			readData := func(out string) (sandboxes.SandboxExecutionResult, string) {
				t.Helper()
				red := sandboxes.CreateRedirectionFiles(box)
				tester.AssertNotError(t, red.CreateNewMetaFileAndRedirect(out+".meta"))
				tester.AssertNotError(t, red.CreateNewStandardOutputFileAndRedirect(out))
				result := box.Execute(*command.GetCommandBuilder("/usr/bin/cat").AddArgs(filepath.Join(dir, "data.txt")), red)
				data, err := sandboxes.ReadBoxFile(box, out, 1<<10)
				tester.AssertNotError(t, err)
				return result, string(data)
			}

			if result, data := readData("before.out"); result.Status == sandboxes.ZERO_EXIT_CODE || data != "" {
				t.Fatalf("box read %s before it was allowed: %+v, %q", dir, result, data)
			}
			tester.AssertNotError(t, box.AddAllowedDirectory(dir))
			if result, data := readData("after.out"); result.Status != sandboxes.ZERO_EXIT_CODE || data != "allowed-dir-canary\n" {
				t.Fatalf("box could not read allowed %s: %+v, %q", dir, result, data)
			}
		})
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
			ExpectedStatus        sandboxes.SandboxExecutionStatus
			ExpectedMessage       string
		}{
			{
				Title:                 "Compile hello.c and run",
				CodeToCompileFilepath: "../../tests/c_test/hello.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
				SecondRunCommand:      *command.GetCommandBuilder("./hello"),
				ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile hello.c",
				CodeToCompileFilepath: "../../tests/c_test/hello.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile empty.c",
				CodeToCompileFilepath: "../../tests/c_test/uncompileable/empty.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("empty.c").AddArgs("-o").AddArgs("empty"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile infiniterecursion.c and run",
				CodeToCompileFilepath: "../../tests/c_test/runtimeerror/infiniterecursion.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniterecursion.c").AddArgs("-o").AddArgs("infinite"),
				SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
				ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 9",
			},
			{
				Title:                 "Compile noinclude.c",
				CodeToCompileFilepath: "../../tests/c_test/uncompileable/noinclude.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("noinclude.c").AddArgs("-o").AddArgs("noinclude"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile nullpointer.c and run",
				CodeToCompileFilepath: "../../tests/c_test/runtimeerror/nullpointer.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("nullpointer.c").AddArgs("-o").AddArgs("nullpointer"),
				SecondRunCommand:      *command.GetCommandBuilder("./nullpointer"),
				ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 11",
			},
			{
				Title:                 "Compile abort.c and run",
				CodeToCompileFilepath: "../../tests/c_test/runtimeerror/abort.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("abort.c").AddArgs("-o").AddArgs("abort"),
				SecondRunCommand:      *command.GetCommandBuilder("./abort"),
				ExpectedStatus:        sandboxes.KILLED_ON_SIGNAL,
				ExpectedMessage:       "Caught fatal signal 6",
			},
			{
				Title:                 "Compile syntaxerror.c",
				CodeToCompileFilepath: "../../tests/c_test/uncompileable/syntaxerror.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("syntaxerror.c").AddArgs("-o").AddArgs("syntaxerror"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile typemismatch.c",
				CodeToCompileFilepath: "../../tests/c_test/uncompileable/typemismatch.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("typemismatch.c").AddArgs("-o").AddArgs("typemismatch"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.ZERO_EXIT_CODE,
				ExpectedMessage:       "",
			},
			{
				Title:                 "Compile unfoundfunc.c",
				CodeToCompileFilepath: "../../tests/c_test/uncompileable/unfoundfunc.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("unfoundfunc.c").AddArgs("-o").AddArgs("unfoundfunc"),
				SecondRunCommand:      emptyCommand,
				ExpectedStatus:        sandboxes.NONZERO_EXIT_CODE,
				ExpectedMessage:       "Exited with error status 1",
			},
			{
				Title:                 "Compile infiniteloop.c and run",
				CodeToCompileFilepath: "../../tests/c_test/runtimeerror/infiniteloop.c",
				RunCommand:            *command.GetCommandBuilder("/usr/bin/gcc").AddArgs("infiniteloop.c").AddArgs("-o").AddArgs("infinite"),
				SecondRunCommand:      *command.GetCommandBuilder("./infinite"),
				ExpectedStatus:        sandboxes.TIMED_OUT,
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

				red := sandboxes.CreateRedirectionFiles(sbx)
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

				res := sbx.Execute(test.RunCommand, red)

				if err != nil {
					t.Fatal(err)
				}

				if !checkEmptyCommand(test.SecondRunCommand) {
					secondRed := sandboxes.CreateRedirectionFiles(sbx)
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

					res = sbx.Execute(test.SecondRunCommand, secondRed)
				}

				if res.Status != test.ExpectedStatus || res.Message != test.ExpectedMessage {
					t.Errorf("expected status %s and message %s, instead got %+v", test.ExpectedStatus, test.ExpectedMessage, res)
				}
			})
		}
	})
}
