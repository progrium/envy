package envy

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fsouza/go-dockerclient"
	"github.com/termie/go-shutil"
)

var (
	dockerEndpoint = "unix:///var/run/docker.sock"
)

func run(cmd *exec.Cmd) int {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if stat, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return int(stat.ExitStatus())
			} else {
				assert(err)
			}
		}
	}
	return 0
}

func assert(err error) {
	if err != nil {
		panic(err) // TODO: replace me
	}
}

func exists(path ...string) bool {
	_, err := os.Stat(filepath.Join(path...))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	assert(err)
	return true
}

func readFile(path string) string {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.Trim(string(data), "\n ")
}

func normalizeLine(s string) string {
	return strings.Trim(s, "\n") + "\n"
}

func writeFile(path, data string) {
	assert(ioutil.WriteFile(path, []byte(normalizeLine(data)), 0644))
}

func appendFile(path, data string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	assert(err)
	defer f.Close()
	_, err = f.WriteString(normalizeLine(data))
	assert(err)
}

func grepFile(path, line string) bool {
	for _, l := range strings.Split(readFile(path), "\n") {
		if l == line {
			return true
		}
	}
	return false
}

func mkdirAll(path ...string) {
	assert(os.MkdirAll(filepath.Join(path...), 0777))
}

func copy(src, dst string) {
	s, err := os.Open(src)
	assert(err)
	defer s.Close()
	d, err := os.Create(dst)
	assert(err)
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		assert(err)
	}
	assert(d.Close())
	fi, err := os.Stat(src)
	assert(err)
	assert(os.Chmod(dst, fi.Mode()))
}

func copyTree(src, dst string) {
	assert(shutil.CopyTree(src, dst, nil))
}

func dockerImage(image string) bool {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	images, err := client.ListImages(docker.ListImagesOptions{})
	assert(err)
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if tag == image {
				return true
			}
			repo := strings.Split(tag, ":")
			if repo[0] == image {
				return true
			}
		}
	}
	return false
}

func dockerRunning(container string) bool {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	containers, err := client.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"status": []string{"running"},
		},
	})
	assert(err)
	for _, cntr := range containers {
		if cntr.ID == container {
			return true
		}
		for _, name := range cntr.Names {
			if name[1:] == container {
				return true
			}
		}
	}
	return false
}

func dockerRemove(container string) {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container,
		Force: true,
	})
}

func dockerCommit(container, image string) {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	_, err = client.CommitContainer(docker.CommitContainerOptions{
		Container:  container,
		Repository: image,
	})
	assert(err)
}

func dockerShellCmd(image string) []string {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	img, err := client.InspectImage(image)
	assert(err)
	if img.Config.Cmd != nil || img.Config.Entrypoint != nil {
		return nil
	}
	return []string{"/bin/sh"}
}

func dockerBuild(contextPath, image string, output io.Writer) {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	context := bytes.NewBuffer(nil)
	tarGzip(context, contextPath)
	assert(client.BuildImage(docker.BuildImageOptions{
		Name:         image,
		OutputStream: output,
		InputStream:  context,
	}))
}

func tarGzip(target io.Writer, path string) {
	gw := gzip.NewWriter(target)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	tarFile := func(_path string, tw *tar.Writer, fi os.FileInfo) {
		fr, err := os.Open(_path)
		assert(err)
		defer fr.Close()
		h := new(tar.Header)
		h.Name = _path[len(path):]
		h.Size = fi.Size()
		h.Mode = int64(fi.Mode())
		h.ModTime = fi.ModTime()
		err = tw.WriteHeader(h)
		assert(err)
		_, err = io.Copy(tw, fr)
		assert(err)
	}
	tarDir(path, tw, tarFile)
}

func tarDir(path string, tw *tar.Writer, tarFile func(string, *tar.Writer, os.FileInfo)) {
	dir, err := os.Open(path)
	assert(err)
	defer dir.Close()
	fis, err := dir.Readdir(0)
	assert(err)
	for _, fi := range fis {
		curPath := path + "/" + fi.Name()
		if fi.IsDir() {
			tarDir(curPath, tw, tarFile)
		} else {
			tarFile(curPath, tw, fi)
		}
	}
}

func dockerRunDetached(opts docker.CreateContainerOptions) {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	cntr, err := client.CreateContainer(opts)
	assert(err)
	assert(client.StartContainer(cntr.ID, nil))
}

func dockerRunInteractive(opts docker.CreateContainerOptions, stdin io.Reader, stdout, stderr io.Writer) int {
	client, err := docker.NewClient(dockerEndpoint)
	assert(err)
	cntr, err := client.CreateContainer(opts)
	assert(err)
	go client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    cntr.ID,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
	})
	assert(client.StartContainer(cntr.ID, nil))
	status, err := client.WaitContainer(cntr.ID)
	assert(err)
	return status
}
