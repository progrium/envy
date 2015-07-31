package envy

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var commands []*Command

type Command struct {
	Run func(context *RunContext)

	Name  string
	Args  []string
	Group string
	Short string
	Long  string

	System bool
	Admin  bool
}

func (c *Command) Match(args []string, session *SessionContext) (bool, []string) {
	if c.System && session == nil && c.Name == filepath.Base(os.Args[0]) {
		return true, args
	}
	if len(args) > 0 && c.Group == "" && args[0] == c.Name {
		return true, args[1:]
	}
	if len(args) > 1 && c.Group == args[0] && args[1] == c.Name {
		return true, args[2:]
	}
	return false, nil
}

func (c *Command) PrintUsage(output io.Writer) {
	if c.Runnable() {
		fmt.Fprintf(output, "Usage: envy %s\n\n", c.FullUsage())
	}
	fmt.Fprintln(output, c.Long)
}

func (c *Command) FullName() string {
	if c.Group != "" {
		return fmt.Sprintf("%s %s", c.Group, c.Name)
	}
	return c.Name
}

func (c *Command) FullUsage() string {
	var usage []string
	if c.Group != "" {
		usage = append(usage, c.Group)
	}
	usage = append(usage, c.Name)
	usage = append(usage, c.Args...)
	return strings.Join(usage, " ")
}

func (c *Command) Runnable() bool {
	return c.Run != nil
}

func (c *Command) Listable() bool {
	return c.Short != ""
}

type RunContext struct {
	Command *Command
	Args    []string
	Session *SessionContext
	Stdout  io.Writer
	Stderr  io.Writer
	Stdin   io.Reader
	Exiter  chan int
	Exited  bool
}

type SessionContext struct {
	User    string
	Session string
	Environ string
	Admin   bool
}

func (c *RunContext) Exit(status int) {
	c.Exiter <- status
	c.Exited = true
}

func (c *RunContext) Run(path string, args ...string) int {
	cmd := exec.Command(path, args...)
	cmd.Stdin = c.Stdin
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	err := cmd.Run()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if stat, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return int(stat.ExitStatus())
			} else {
				assert(err)
			}
		}
	}
	return 0
}

func RunCmd(args []string, stdin io.Reader, stdout, stderr io.Writer, exitCh chan int, context *SessionContext) {
	ctx := &RunContext{
		Stdout:  stdout,
		Stderr:  stderr,
		Stdin:   stdin,
		Exiter:  exitCh,
		Session: context,
	}
	for _, cmd := range commands {
		if ok, cmdargs := cmd.Match(args, context); ok && cmd.Runnable() {
			ctx.Command = cmd
			ctx.Args = cmdargs
			cmd.Run(ctx)
			if !ctx.Exited {
				exitCh <- 0
			}
			return
		}
	}
	fmt.Fprintf(stderr, "Unknown command: %s\n", args[:])
	PrintUsage(stderr)
	exitCh <- 2
}
