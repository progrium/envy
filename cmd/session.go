package envy

import (
	"fmt"
	"os"
	"os/exec"
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
		context.Exit(128)
	},
}

var cmdSessionCommit = &Command{
	Short: "commit session changes to environment image",
	Long:  `Commit saves changes made in the session to the environment image.`,

	Group: "session",
	Name:  "commit",
	Args:  []string{"[<environ>]"},
	Run: func(context *RunContext) {
		var environ *Environ
		if context.Arg(0) != "" {
			environ = GetEnviron(context.Session.User.Name, context.Arg(0))
		} else {
			environ = context.Session.Environ()
		}
		fmt.Fprintf(context.Stdout, "Committing to %s ...\n", environ.Name)
		dockerCommit(context.Session.DockerName(), environ.DockerImage())
		context.Exit(128)
	},
}

var cmdSessionSwitch = &Command{
	Short: "switch session to different environment",
	Long:  `Switch reloads session from a new environment image.`,

	Group: "session",
	Name:  "switch",
	Args:  []string{"[<environ>]"},
	Run: func(context *RunContext) {
		if context.Arg(0) == "" {
			return
		}
		context.Session.SetEnviron(context.Arg(0))
		context.Exit(128)
	},
}

type Session struct {
	User *User
	Name string
}

func (s *Session) Environ() *Environ {
	return GetEnviron(s.User.Name, readFile(s.Path("environ")))
}

func (s *Session) SetEnviron(name string) {
	writeFile(s.Path("environ"), name)
}

func (s *Session) Path(parts ...string) string {
	return Envy.Path(append([]string{"users", s.User.Name, "sessions", s.Name}, parts...)...)
}

func (s *Session) DockerName() string {
	return s.Name
}

func (s *Session) Enter(context *RunContext, environ *Environ) int {
	defer s.Cleanup()
	fmt.Fprintln(context.Stdout, "Entering session...")
	s.SetEnviron(environ.Name)
	envySock := startEnvySock(s.Path("run/envy.sock"), s)
	defer envySock.Close()
	for {
		dockerRemove(s.Name)
		environ := s.Environ()
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
			fmt.Sprintf("--volume=%s:/sbin/envy:ro", Envy.HostPath("bin/envy")),
		}
		if s.User.Admin() {
			args = append(args, fmt.Sprintf("--volume=%s:/envy", Envy.HostPath()))
		}
		args = append(args, environ.DockerImage())
		if dockerShellCmd(environ.DockerImage()) != nil {
			args = append(args, dockerShellCmd(environ.DockerImage())...)
		}
		status := context.Run(exec.Command("/bin/docker", args...))
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
	// TODO: clean up sessions without docker running
	for {
		s := user.Session(fmt.Sprintf("%s.%v", user.Name, n))
		if !exists(s.Path()) {
			return s.Name
		}
		n += 1
	}
}
