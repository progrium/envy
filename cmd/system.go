package envy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func init() {
	commands = append(commands,
		cmdEnter,
		cmdAuth,
		cmdServe,
	)
}

var cmdEnter = &Command{
	System: true,
	Name:   "enter",
	Run: func(context *RunContext) {
		user, environ := parseUserEnviron(os.Getenv("USER"))
		if !Envy.Allow(user, environ) {
			fmt.Fprintln(context.Stderr, "User is forbidden.")
			context.Exit(2)
		}
		context.Exit(GetSession(user).Enter(context, GetEnviron(user, environ)))
	},
}

var cmdAuth = &Command{
	System: true,
	Name:   "auth",
	Args:   []string{"<user>", "<key>"},
	Run: func(context *RunContext) {
		if os.Getenv("ENVY_NOAUTH") != "" {
			return
		}
		if len(context.Args) < 2 {
			context.Exit(2)
			return
		}
		user, _ := parseUserEnviron(context.Args[0])
		if !githubKeyAuth(user, context.Args[1]) {
			context.Exit(1)
		}
	},
}

var cmdServe = &Command{
	System: true,
	Name:   "serve",
	Run: func(context *RunContext) {
		Envy.Setup()

		go func() {
			exec.Command("/bin/docker", "pull", "progrium/dind:latest").Run()
		}()

		logger := log.New(context.Stdout, "", 0)
		logger.Fatal(http.ListenAndServe(":80", nil))
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
