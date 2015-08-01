package envy

import (
	"fmt"
	"os/exec"

	"github.com/fsouza/go-dockerclient"
)

func init() {
	commands = append(commands, cmdEnvironRebuild)
}

var cmdEnvironRebuild = &Command{
	Short: "rebuild environment image",
	Long:  `Rebuild does a Docker build with your environment Dockerfile.`,

	Group: "environ",
	Name:  "rebuild",
	Args:  []string{"[--force]"}, // TODO
	Run: func(context *RunContext) {
		environ := context.Session.Environ()
		cmd := exec.Command("/bin/docker", "build", "-t", environ.DockerImage(), ".")
		cmd.Dir = environ.Path()
		context.Stdin = nil
		context.Run(cmd)
		context.Exit(128)
	},
}

type Environ struct {
	User *User
	Name string
}

func (e *Environ) Path(parts ...string) string {
	return Envy.Path(append([]string{"users", e.User.Name, "environs", e.Name}, parts...)...)
}

func (e *Environ) DockerImage() string {
	return fmt.Sprintf("%s/%s", e.User.Name, e.Name)
}

func (e *Environ) DockerName() string {
	return fmt.Sprintf("%s.%s", e.User.Name, e.Name)
}

func GetEnviron(user, name string) *Environ {
	e := &Environ{
		Name: name,
		User: GetUser(user),
	}
	if !exists(e.Path()) {
		copyTree(Envy.DataPath("environ"), e.Path())
	}
	mkdirAll(e.Path("run"))
	if !dockerRunning(e.DockerName()) {
		dockerRemove(e.DockerName())
		dockerRunDetached(docker.CreateContainerOptions{
			Name: e.DockerName(),
			Config: &docker.Config{
				Hostname: e.Name,
				Image:    "progrium/dind:latest",
			},
			HostConfig: &docker.HostConfig{
				Privileged:    true,
				RestartPolicy: docker.RestartPolicy{Name: "always"},
				Binds: []string{
					fmt.Sprintf("%s:/var/run", Envy.HostPath(e.Path("run"))),
				},
			},
		})
	}
	if !dockerImage(e.DockerImage()) {
		cmd := exec.Command("/bin/docker", "build", "-t", e.DockerImage(), ".")
		cmd.Dir = e.Path()
		assert(cmd.Run())
	}
	return e
}
