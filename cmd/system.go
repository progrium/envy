package envy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func CheckSystemCmd() {
	cmd := filepath.Base(os.Args[0])
	systemCmds := []string{"enter", "auth", "serve"}
	for i := range systemCmds {
		if cmd == systemCmds[i] {
			os.Args = append([]string{os.Args[0], cmd}, os.Args[1:]...)
			Cmd.AddCommand(cmdEnter)
			Cmd.AddCommand(cmdAuth)
			Cmd.AddCommand(cmdServe)
			return
		}
	}
}

var cmdEnter = &cobra.Command{
	Use: "enter",
	Run: func(cmd *cobra.Command, args []string) {
		user, environ := parseUserEnviron(os.Getenv("USER"))
		if !Envy.Allow(user, environ) {
			fmt.Fprintln(os.Stderr, "User is forbidden.")
			os.Exit(2)
		}
		os.Exit(NewSession(user).Enter(GetEnviron(user, environ)))
	},
}

var cmdAuth = &cobra.Command{
	Use: "auth <user> <key>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 2 {
			os.Exit(2)
			return
		}
		if os.Getenv("ENVY_NOAUTH") != "" {
			log.Println("Warning: ENVY_NOAUTH is set allowing all SSH connections")
			return
		}
		user, _ := parseUserEnviron(args[0])
		if !githubKeyAuth(user, args[1]) {
			log.Println("auth[ssh]: not allowing", user)
			os.Exit(1)
		}
		log.Println("auth[ssh]: allowing", user)
	},
}

var cmdServe = &cobra.Command{
	Use: "serve",
	Run: func(cmd *cobra.Command, args []string) {
		log.Println("Setting up Envy root ...")
		Envy.Setup()

		go func() {
			log.Println("Pulling progrium/dind:latest ...")
			exec.Command("/bin/docker", "pull", "progrium/dind:latest").Run()
		}()

		log.Println("Starting HTTP server on 80 ...")
		log.Fatal(http.ListenAndServe(":80", nil))
	},
}

func parseUserEnviron(in string) (string, string) {
	parts := strings.Split(in, "+")
	user := parts[0]
	var env string
	if len(parts) > 1 {
		env = parts[1]
	} else {
		env = user
	}
	return user, env
}

func githubUserAuth(user, passwd string) bool {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "https://api.github.com", nil)
	req.SetBasicAuth(user, passwd)
	resp, _ := client.Do(req)
	return resp.StatusCode == 200
}

func githubKeyAuth(user, key string) bool {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://github.com/%s.keys", user), nil)
	resp, _ := client.Do(req)
	if resp.StatusCode != 200 {
		return false
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), key)
}
