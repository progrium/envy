package envy

import (
	"bytes"
	"fmt"

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
	Args:  []string{"[--force]"},
	Run: func(context *RunContext) {
		fmt.Fprintln(context.Stdout, "rebuilding!")
		// docker build -t "$USER/$ENVIRON" .
	},
}

type Environ struct {
	User *User
	Name string
}

func (e *Environ) Path(parts ...string) string {
	return Envy.Path(append([]string{"users", e.User.Name, "environs", e.Name}, parts...)...)
}

func (e *Environ) Image() string {
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
	if !dockerImage(e.Image()) {
		dockerBuild(e.Path(), e.Image(), new(bytes.Buffer))
	}
	return e
}
