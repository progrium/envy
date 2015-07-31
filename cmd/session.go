package envy

import (
	"fmt"
	"os"
	"time"
)

func init() {
	commands = append(commands,
		cmdSessionReload,
		cmdSessionCommit,
		cmdSessionSwitch,
	)
}

var cmdSessionReload = &Command{
	Short: "reload session from environment image",
	Long:  `Reload recreates the current session container from the environment image.`,

	Group: "session",
	Name:  "reload",
	Run: func(context *RunContext) {
		if context.Session != nil {
			context.Exit(128)
		}
	},
}

var cmdSessionCommit = &Command{
	Short: "commit session state to environment image",
	Long:  `Commit saves changes made in the session to the environment image.`,

	Group: "session",
	Name:  "commit",
	Args:  []string{"[<environ>]"},
	Run: func(context *RunContext) {
		fmt.Fprintln(context.Stdout, "commit!")
		//echo "Committing to ${args:-$USER/$ENVIRON} ... "
		//docker commit "$session" "${args:-$USER/$ENVIRON}" > /dev/null
	},
}

var cmdSessionSwitch = &Command{
	//Short: "commit session state to environment image",
	//Long:  `Commit saves changes made in the session to the environment image.`,

	Group: "session",
	Name:  "switch",
	Args:  []string{"[<environ>]"},
	Run: func(context *RunContext) {
		fmt.Fprintln(context.Stdout, "switch!")
	},
}

type Session struct {
	User *User
	Name string
}

func (s *Session) Path(parts ...string) string {
	return Envy.Path(append([]string{"users", s.User.Name, "sessions", s.Name}, parts...)...)
}

func (s *Session) Enter(context *RunContext, environ *Environ) int {
	defer s.Cleanup()
	fmt.Fprintln(context.Stdout, "Entering session...")
	envySock := startEnvySock(s.Path("run/envy.sock"), &SessionContext{
		User:    s.User.Name,
		Session: s.Name,
		Environ: environ.Name,
		Admin:   true,
	})
	defer envySock.Close()
	for {
		// reload commands?
		dockerRemove(s.Name)
		args := []string{"run", "-it",
			fmt.Sprintf("--name=%s", s.Name),
			fmt.Sprintf("--net=container:%s", environ.DockerName()),

			fmt.Sprintf("--env=HOSTNAME=%s", environ.Name),
			fmt.Sprintf("--env=ENVY_RELOAD=%v", int32(time.Now().Unix())),
			fmt.Sprintf("--env=ENVY_SESSION=%s", s.Name),
			fmt.Sprintf("--env=ENVY_USER=%s", s.User.Name),
			"--env=DOCKER_HOST=unix:///var/run/docker.sock",
			"--env=ENV=/etc/envyrc",

			fmt.Sprintf("--volume=%s:/var/run/docker.sock", Envy.HostPath(environ.Path("run/docker.sock"))),
			fmt.Sprintf("--volume=%s:/var/run/envy.sock:ro", Envy.HostPath(s.Path("run/envy.sock"))),
			fmt.Sprintf("--volume=%s:/etc/envyrc:ro", Envy.HostPath(environ.Path("envyrc"))),
			fmt.Sprintf("--volume=%s:/root/environ", Envy.HostPath(environ.Path())),
			fmt.Sprintf("--volume=%s:/root", Envy.HostPath(s.User.Path("root"))),
			fmt.Sprintf("--volume=%s:/home/%s", Envy.HostPath(s.User.Path("home")), s.User.Name),
			fmt.Sprintf("--volume=%s:/envy", Envy.HostPath()), // TODO: admin only
			fmt.Sprintf("--volume=%s:/sbin/envy:ro", Envy.HostPath("bin/envy")),

			environ.Image(),
		}
		if dockerShellCmd(environ.Image()) != nil {
			args = append(args, dockerShellCmd(environ.Image())...)
		}
		status := context.Run("/bin/docker", args...)
		if status != 128 {
			return status
		}
	}
}

func (s *Session) Cleanup() {
	dockerRemove(s.Name)
	os.Remove(s.Path("run/envy.sock"))
}

func GetSession(user string) *Session {
	u := GetUser(user)
	s := &Session{
		Name: nextSessionName(u),
		User: u,
	}
	mkdirAll(s.Path("run"))
	return s
}

func nextSessionName(user *User) string {
	n := 0
	// TODO: panic on max n
	for {
		s := user.Session(fmt.Sprintf("%s.%v", user.Name, n))
		if !exists(s.Path()) {
			return s.Name
		}
		n += 1
	}
}
