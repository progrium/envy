package envy

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	cmdAdmin.AddCommand(cmdAdminList)
	cmdAdmin.AddCommand(cmdAdminRemove)
	cmdAdmin.AddCommand(cmdAdminAdd)
}

func CheckAdminCmd() {
	if GetUser(os.Getenv("ENVY_USER")).Admin() {
		Cmd.AddCommand(cmdAdmin)
	}
}

var cmdAdmin = &cobra.Command{
	Use: "admin",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var cmdAdminList = &cobra.Command{
	Short: "list admin users",
	Long:  `Lists users in the admins ACL file.`,

	Use: "ls",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stdout, readFile(Envy.Path("config/admins")))
	},
}

var cmdAdminRemove = &cobra.Command{
	Short: "remove admin user",
	Long:  `Removes user from the admins ACL file.`,

	Use: "rm <user>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(1)
		}
		var admins []string
		adminConfig := readFile(Envy.Path("config/admins"))
		for _, admin := range strings.Split(adminConfig, "\n") {
			if admin != args[0] {
				admins = append(admins, admin)
			}
		}
		writeFile(Envy.Path("config/admins"), strings.Join(admins, "\n"))
	},
}

var cmdAdminAdd = &cobra.Command{
	Short: "add admin user",
	Long:  `Adds user to the admins ACL file.`,

	Use: "add <user>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(1)
		}
		if grepFile(Envy.Path("config/admins"), args[0]) {
			return
		}
		appendFile(Envy.Path("config/admins"), args[0])
	},
}
