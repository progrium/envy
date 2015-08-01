package envy

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

var EnvySocket = "/var/run/envy.sock"

func DetectSocket() bool {
	probablyEnvy := os.Getenv("ENVY_SESSION")
	if !exists(EnvySocket) && probablyEnvy != "" {
		fmt.Fprintln(os.Stderr, "Unable to find Envy socket.")
		os.Exit(1)
	}
	return exists(EnvySocket)
}

func SocketClient(args []string) {
	client, err := ssh.Dial("unix", EnvySocket, new(ssh.ClientConfig))
	assert(err)
	session, err := client.NewSession()
	assert(err)
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin
	err = session.Run(strings.Join(args, " "))
	session.Close()
	if err != nil {
		if exiterr, ok := err.(*ssh.ExitError); ok {
			os.Exit(exiterr.ExitStatus())
		} else {
			assert(err)
		}
	}
}

func startEnvySock(path string, session *Session) net.Listener {
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	assert(err)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				break
			}
			go handleSSHConn(conn, session)
		}
	}()
	return ln
}

func handleSSHConn(conn net.Conn, session *Session) {
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
		go handleSSHChannel(ch, session)
	}
}

func handleSSHChannel(newChan ssh.NewChannel, session *Session) {
	ch, reqs, err := newChan.Accept()
	if err != nil {
		log.Println("handle channel failed:", err)
		return
	}
	exitCh := make(chan int)
	go func() {
		status := struct{ Status uint32 }{uint32(<-exitCh)}
		_, err = ch.SendRequest("exit-status", false, ssh.Marshal(&status))
		assert(err)
		ch.Close()
	}()
	for req := range reqs {
		go func(req *ssh.Request) {
			if req.WantReply {
				req.Reply(true, nil)
			}
			switch req.Type {
			case "exec":
				var payload = struct{ Value string }{}
				ssh.Unmarshal(req.Payload, &payload)
				line := strings.Trim(payload.Value, "\n")
				var args []string
				if line != "" {
					args = strings.Split(line, " ")
				}
				RunCmd(args, ch, ch, ch.Stderr(), exitCh, session)
			}
		}(req)
	}
}
