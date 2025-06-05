package cmdbuilder

import (
	"strings"

	"github.com/hashicorp/go-version"
)

type CommandArgumentOption func(*Command, string) error

type Command struct {
	Base    BaseCommand
	Args    []string
	Version *version.Version
}

func WithConditionalFlags(c *Command, subcommand string, options ...CommandArgumentOption) error {
	for _, option := range options {
		if err := option(c, subcommand); err != nil {
			return err
		}
	}
	return nil
}

func MustVersionConstraint(versionConstraint string) *version.Constraints {
	constraint, err := version.NewConstraint(versionConstraint)
	if err != nil {
		panic(err)
	}
	return &constraint
}

func AppendFlag(commandName string, subCommand string, flag string, versionConstraint *version.Constraints) CommandArgumentOption {
	return func(c *Command, curSubCommand string) error {
		if c.Base.Equal(commandName) && curSubCommand == subCommand {
			if versionConstraint == nil || versionConstraint.Check(c.Version) {
				c.Args = append(c.Args, flag)
			}
		}
		return nil
	}
}

func PrependFlag(commandName string, subCommand string, flag string, versionConstraint *version.Constraints) CommandArgumentOption {
	return func(c *Command, curSubCommand string) error {
		if c.Base.Equal(commandName) && curSubCommand == subCommand {
			if versionConstraint == nil || versionConstraint.Check(c.Version) {
				c.Args = append([]string{flag}, c.Args...)
			}
		}
		return nil
	}
}

func RemoveFlag(commandName string, subCommand string, flag string, versionConstraint *version.Constraints) CommandArgumentOption {
	return func(c *Command, curSubCommand string) error {
		if c.Base.Equal(commandName) && curSubCommand == subCommand {
			if versionConstraint == nil || versionConstraint.Check(c.Version) {
				c.Args = filter(c.Args, func(s string) bool {
					return s != flag
				})
			}
		}
		return nil
	}
}

func NewBaseCommand(name string, args ...string) BaseCommand {
	return BaseCommand{
		name: name,
		args: args,
	}
}

type BaseCommand struct {
	name string
	args []string
}

func (c *BaseCommand) Name() string {
	return c.name
}

func (c *BaseCommand) Equal(v string) bool {
	return c.String() == v
}

func (c *BaseCommand) String() string {
	return strings.Join(append([]string{c.name}, c.args...), " ")
}

func (c *BaseCommand) Args(args ...string) []string {
	return append(c.args, args...)
}

func filter(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}
