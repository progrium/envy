package envy

import (
	"fmt"
	"strings"
)

func init() {
	commands = append(commands,
		cmdAdminList,
		cmdAdminRemove,
		cmdAdminAdd,
	)
}

var cmdAdminList = &Command{
	Short: "list admin users",
	Long:  `Lists users in the admins ACL file.`,
	Admin: true,

	Group: "admin",
	Name:  "ls",
	Run: func(context *RunContext) {
		fmt.Fprintln(context.Stdout, readFile(Envy.Path("config/admins")))
	},
}

var cmdAdminRemove = &Command{
	Short: "remove admin user",
	Long:  `Removes user from the admins ACL file.`,
	Admin: true,

	Group: "admin",
	Name:  "rm",
	Args:  []string{"<user>"},
	Run: func(context *RunContext) {
		if context.Arg(0) == "" {
			context.Exit(1)
			return
		}
		var admins []string
		adminConfig := readFile(Envy.Path("config/admins"))
		for _, admin := range strings.Split(adminConfig, "\n") {
			if admin != context.Arg(0) {
				admins = append(admins, admin)
			}
		}
		writeFile(Envy.Path("config/admins"), strings.Join(admins, "\n"))
	},
}

var cmdAdminAdd = &Command{
	Short: "add admin user",
	Long:  `Adds user to the admins ACL file.`,
	Admin: true,

	Group: "admin",
	Name:  "add",
	Args:  []string{"<user>"},
	Run: func(context *RunContext) {
		if context.Arg(0) == "" {
			context.Exit(1)
			return
		}
		if grepFile(Envy.Path("config/admins"), context.Arg(0)) {
			return
		}
		appendFile(Envy.Path("config/admins"), context.Arg(0))
	},
}
