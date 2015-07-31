package envy

import (
	"os"
	"path/filepath"
	"strings"
)

var Envy = new(EnvyRoot).Setup()

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

func (r *EnvyRoot) Setup() *EnvyRoot {
	if os.Args[0] != "/bin/envy" {
		// not server
		return nil
	}
	mkdirAll(r.Path("users"))
	mkdirAll(r.Path("bin"))
	os.RemoveAll(r.Path("bin/envy"))
	copy("/bin/envy", r.Path("bin/envy"))
	return r
}
