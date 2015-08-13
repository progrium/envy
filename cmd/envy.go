package envy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var Envy = new(EnvyRoot)

var Cmd = &cobra.Command{
	Use:   "envy",
	Short: "",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Usage()
	},
}

type EnvyRoot struct{}

func (r *EnvyRoot) Path(parts ...string) string {
	return filepath.Join(append([]string{"/envy"}, parts...)...)
}

func (r *EnvyRoot) HostPath(parts ...string) string {
	return filepath.Join(os.Getenv("HOST_ROOT"), strings.TrimPrefix(filepath.Join(parts...), "/envy"))
}

func (r *EnvyRoot) DataPath(parts ...string) string {
	path := append([]string{"/tmp/data"}, parts...)
	return filepath.Join(path...)
}

func (r *EnvyRoot) Allow(user, environ string) bool {
	if !r.checkUserAcl(user) {
		return false
	}
	parts := strings.Split(environ, "/")
	if len(parts) > 1 {
		// TODO: shared environ acl
		return false
	}
	return true
}

func (r *EnvyRoot) checkUserAcl(user string) bool {
	if readFile(r.Path("config/users")) == "*" {
		return true
	}
	return grepFile(r.Path("config/users"), user)
}

func (r *EnvyRoot) Setup() {
	mkdirAll(r.Path("users"))
	mkdirAll(r.Path("config"))
	mkdirAll(r.Path("bin"))
	if !exists(r.Path("config/users")) {
		writeFile(r.Path("config/users"), "*")
	}
	os.RemoveAll(r.Path("bin/envy"))
	os.RemoveAll(r.Path("bin/docker"))
	copy("/bin/envy", r.Path("bin/envy"))
	copy("/bin/docker", r.Path("bin/docker"))
}
