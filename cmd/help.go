package envy

import (
	"fmt"
	"html/template"
	"io"
	"sort"
	"text/tabwriter"
)

func init() {
	commands = append(commands,
		cmdHelp,
		helpCommands,
	)
}

var cmdHelp = &Command{
	Long: `Help shows usage for a command or other topic.`,

	Name: "help",
	Args: []string{"[<topic>]"},
	Run: func(context *RunContext) {
		args := context.Args
		if len(args) == 0 {
			printUsage(context.Stderr, context.Session)
			return // not os.Exit(2); success
		}
		switch args[0] {
		case helpCommands.Name:
			printAllUsage(context.Stderr)
			return
		}

		for _, cmd := range commands {
			if ok, _ := cmd.Match(args, context.Session); ok {
				cmd.PrintUsage(context.Stderr)
				return
			}
		}

		fmt.Fprintf(context.Stderr, "Unknown help topic: %q. Run 'envy help'.\n", args[0])
		context.Exit(2)
	},
}

var helpCommands = &Command{
	Name:  "commands",
	Short: "list all commands with usage",
	Long:  "(not displayed; see special case in runHelp)",
}

func maxStrLen(strs []string) (strlen int) {
	for i := range strs {
		if len(strs[i]) > strlen {
			strlen = len(strs[i])
		}
	}
	return
}

var usageTemplate = template.Must(template.New("usage").Parse(`
Usage: envy <command> [options] [arguments]

Commands:
{{range .Commands}}
    {{.FullName | printf (print "%-" $.MaxRunListName "s")}}  {{.Short}}{{end}}

Run 'envy help [command]' for details.
`[1:]))

func printUsage(output io.Writer, session *Session) {
	var runListNames []string
	var runListCmds []*Command
	for i := range commands {
		if commands[i].Runnable() && commands[i].Listable(session) {
			runListNames = append(runListNames, commands[i].FullName())
			runListCmds = append(runListCmds, commands[i])
		}
	}

	usageTemplate.Execute(output, struct {
		Commands       []*Command
		MaxRunListName int
	}{
		runListCmds,
		maxStrLen(runListNames),
	})
}

func printAllUsage(output io.Writer) {
	w := tabwriter.NewWriter(output, 1, 2, 2, ' ', 0)
	defer w.Flush()
	cl := commandList(commands)
	sort.Sort(cl)
	for i := range cl {
		if cl[i].Runnable() {
			listRec(w, "envy "+cl[i].FullUsage(), "# "+cl[i].Short)
		}
	}
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}

type commandList []*Command

func (cl commandList) Len() int           { return len(cl) }
func (cl commandList) Swap(i, j int)      { cl[i], cl[j] = cl[j], cl[i] }
func (cl commandList) Less(i, j int) bool { return cl[i].Name < cl[j].Name }

type commandMap map[string]commandList
