package command

import (
	"strings"
)

type CommandBuilder struct {
	Program string
	Args    []string
}

func GetCommandBuilder(program string) *CommandBuilder {
	return &CommandBuilder{
		Program: program,
		Args:    []string{},
	}
}

func (cb *CommandBuilder) AddArgs(args ...string) *CommandBuilder {
	cb.Args = append(cb.Args, args...)
	return cb
}

func (cb *CommandBuilder) BuildArgs() string {
	return strings.Join(cb.Args, " ")
}

func (cb *CommandBuilder) BuildFullCommand() string {
	return cb.Program + " " + cb.BuildArgs()
}
