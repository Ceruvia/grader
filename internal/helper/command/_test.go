package command_test

import (
	"testing"

	"github.com/Ceruvia/grader/internal/command"
	"github.com/Ceruvia/grader/internal/utils"
)

func TestGetCommandBuilder(t *testing.T) {
	t.Run("it should be able to create a CommandBuilder", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc")
		want := &command.CommandBuilder{
			Program: "gcc",
			Args:    []string{},
		}

		utils.AssertDeep(t, got, want)
	})
}

func TestAddArgs(t *testing.T) {
	t.Run("it should be able to add an argument to empty command", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc").AddArgs("hello.c")
		want := &command.CommandBuilder{
			Program: "gcc",
			Args:    []string{"hello.c"},
		}

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should be able to chain arguments to command", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc").AddArgs("hello.c").AddArgs("-o").AddArgs("hello")
		want := &command.CommandBuilder{
			Program: "gcc",
			Args:    []string{"hello.c", "-o", "hello"},
		}

		utils.AssertDeep(t, got, want)
	})

	t.Run("it should be able to accept variadic arguments", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc").AddArgs("hello.c", "-o", "hello")
		want := &command.CommandBuilder{
			Program: "gcc",
			Args:    []string{"hello.c", "-o", "hello"},
		}

		utils.AssertDeep(t, got, want)
	})
}

func TestBuildArgs(t *testing.T) {
	t.Run("it should be able to generate args string", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc").AddArgs("hello.c", "-o", "hello").BuildArgs()
		want := "hello.c -o hello"

		utils.AssertDeep(t, got, want)
	})
}

func TestBuildFullCommand(t *testing.T) {
	t.Run("it should be able to generate full command string", func(t *testing.T) {
		got := command.GetCommandBuilder("gcc").AddArgs("hello.c", "-o", "hello").BuildFullCommand()
		want := "gcc hello.c -o hello"

		utils.AssertDeep(t, got, want)
	})
}
