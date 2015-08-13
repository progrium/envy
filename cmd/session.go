package envy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func init() {
	cmdSession.AddCommand(cmdSessionReload)
	cmdSession.AddCommand(cmdSessionCommit)
	cmdSession.AddCommand(cmdSessionSwitch)
	Cmd.AddCommand(cmdSession)
}

var cmdSession = &cobra.Command{
	Use: "session",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var cmdSessionReload = &cobra.Command{
	Short: "reload session from environment image",
	Long:  `Reload recreates the current session container from the environment image.`,

	Use: "reload",
	Run: func(cmd *cobra.Command, args []string) {
		session := GetSession(os.Getenv("ENVY_USER"), os.Getenv("ENVY_SESSION"))
		log.Println(session.User.Name, "| reloading session", session.Name)
		os.Exit(128)
	},
}

var cmdSessionCommit = &cobra.Command{
	Short: "commit session changes to environment image",
	Long:  `Commit saves changes made in the session to the environment image.`,

	Use: "commit [<environ>]",
	Run: func(cmd *cobra.Command, args []string) {
		session := GetSession(os.Getenv("ENVY_USER"), os.Getenv("ENVY_SESSION"))
		var environ *Environ
		if len(args) > 0 {
			environ = GetEnviron(os.Getenv("ENVY_USER"), args[0])
		} else {
			environ = session.Environ()
		}
		log.Println(session.User.Name, "| committing session", session.Name, "to", environ.Name)
		fmt.Fprintf(os.Stdout, "Committing to %s ...\n", environ.Name)
		dockerCommit(session.DockerName(), environ.DockerImage())
		os.Exit(128)
	},
}

var cmdSessionSwitch = &cobra.Command{
	Short: "switch session to different environment",
	Long:  `Switch reloads session from a new environment image.`,

	Use: "switch <environ>",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			return
		}
		session := GetSession(os.Getenv("ENVY_USER"), os.Getenv("ENVY_SESSION"))
		session.SetEnviron(args[0])
		log.Println(session.User.Name, "| switching session", session.Name, "to", args[0])
		os.Exit(128)
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

func (s *Session) Enter(environ *Environ) int {
	defer s.Cleanup()
	log.Println(s.User.Name, "| entering session", s.Name)
	os.Setenv("ENVY_USER", s.User.Name)
	os.Setenv("ENVY_SESSION", s.Name)
	s.SetEnviron(environ.Name)
	fmt.Fprintln(os.Stdout, "Entering session...")
	envySock := startSessionServer(s.Path("run/envy.sock"))
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
		status := run(exec.Command("/bin/docker", args...))
		if status != 128 {
			return status
		}
	}
}

func (s *Session) Cleanup() {
	log.Println("Cleaning up")
	dockerRemove(s.Name)
	os.Remove(s.Path("run/envy.sock"))
}

func NewSession(user string) *Session {
	return GetSession(user, nextSessionName(GetUser(user)))
}

func GetSession(user, name string) *Session {
	u := GetUser(user)
	s := &Session{
		Name: name,
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

func startSessionServer(path string) net.Listener {
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	assert(err)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			go handleSSHConn(conn)
		}
	}()
	return ln
}

func handleSSHConn(conn net.Conn) {
	defer conn.Close()
	config := &ssh.ServerConfig{NoClientAuth: true}
	privateBytes, err := ioutil.ReadFile(Envy.DataPath("id_host"))
	assert(err)
	private, err := ssh.ParsePrivateKey(privateBytes)
	assert(err)
	config.AddHostKey(private)
	_, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Println(err)
		return
	}
	go ssh.DiscardRequests(reqs)
	for ch := range chans {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}
		go handleSSHChannel(ch)
	}
}

func handleSSHChannel(newChan ssh.NewChannel) {
	ch, reqs, err := newChan.Accept()
	if err != nil {
		log.Println("handle channel failed:", err)
		return
	}
	for req := range reqs {
		go func(req *ssh.Request) {
			if req.WantReply {
				req.Reply(true, nil)
			}
			switch req.Type {
			case "exec":
				defer ch.Close()
				var payload = struct{ Value string }{}
				ssh.Unmarshal(req.Payload, &payload)
				line := strings.Trim(payload.Value, "\n")
				var args []string
				if line != "" {
					args = strings.Split(line, " ")
				}
				cmd := exec.Command("/bin/envy", args...)
				cmd.Stdout = ch
				cmd.Stderr = ch.Stderr()
				err := cmd.Run()
				status := struct{ Status uint32 }{0}
				if err != nil {
					if exiterr, ok := err.(*exec.ExitError); ok {
						if stat, ok := exiterr.Sys().(syscall.WaitStatus); ok {
							status = struct{ Status uint32 }{uint32(stat.ExitStatus())}
						} else {
							assert(err)
						}
					}
				}
				_, err = ch.SendRequest("exit-status", false, ssh.Marshal(&status))
				assert(err)
				return
			}
		}(req)
	}
}
