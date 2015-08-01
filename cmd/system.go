package envy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/progrium/envy/pkg/hterm"
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

		logger := log.New(context.Stdout, "", 0)
		http.HandleFunc("/u/", func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) < 3 {
				http.NotFound(w, r)
				return
			}
			pathUser := parts[2]
			var pathEnv, sshUser string
			if len(parts) > 3 && parts[3] != "hterm" {
				pathEnv = parts[3]
				sshUser = pathUser + "+" + pathEnv
			} else {
				sshUser = pathUser
			}
			// passthrough auth for hterm. use cookie to do this right
			if !strings.Contains(r.URL.Path, "hterm") {
				user, passwd, ok := r.BasicAuth()
				if !ok || user != pathUser || !githubUserAuth(user, passwd) {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", pathUser))
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
			}
			w.Header().Set("Hterm-Title", "Envy Term")
			hterm.Handle(w, r, func(args string) *hterm.Pty {
				cmd := exec.Command("/bin/enter", parts[2])
				cmd.Env = os.Environ()
				cmd.Env = append(cmd.Env, fmt.Sprintf("USER=%s", sshUser))
				pty, err := hterm.NewPty(cmd)
				if err != nil {
					log.Fatal(err)
				}
				return pty
			})
		})
		logger.Println("starting http server on :80 ...")
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
