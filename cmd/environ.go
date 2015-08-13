package envy

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/fsouza/go-dockerclient"
	"github.com/spf13/cobra"
)

func init() {
	cmdEnviron.AddCommand(cmdEnvironRebuild)
	Cmd.AddCommand(cmdEnviron)
}

var cmdEnviron = &cobra.Command{
	Use: "environ",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

var cmdEnvironRebuild = &cobra.Command{
	Short: "rebuild environment image",
	Long:  `Rebuild does a Docker build with your environment Dockerfile.`,

	Use: "rebuild [--force]", // TODO
	Run: func(_ *cobra.Command, args []string) {
		session := GetSession(os.Getenv("ENVY_USER"), os.Getenv("ENVY_SESSION"))
		environ := session.Environ()
		log.Println(session.User.Name, "| rebuilding environ", environ.Name)
		cmd := exec.Command("/bin/docker", "build", "-t", environ.DockerImage(), ".")
		cmd.Dir = environ.Path()
		run(cmd)
		os.Exit(128)
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
		log.Println(user, "| starting dind for environ", e.Name)
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
					fmt.Sprintf("%s:/usr/bin/docker", Envy.HostPath("bin/docker")),
					fmt.Sprintf("%s:/var/run", Envy.HostPath(e.Path("run"))),
				},
			},
		})
	}
	if !dockerImage(e.DockerImage()) {
		log.Println(user, "| building environ", e.Name)
		cmd := exec.Command("/bin/docker", "build", "-t", e.DockerImage(), ".")
		cmd.Dir = e.Path()
		assert(cmd.Run())
	}
	return e
}
