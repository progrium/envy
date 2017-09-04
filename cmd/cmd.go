package envy

import (
	"bufio"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

var EnvySocket = "/var/run/envy.sock"

func ClientMode() bool {
	return exists(EnvySocket)
}

func RunClient(args []string) {
	log.SetFlags(0)
	client, err := ssh.Dial("unix", EnvySocket, &ssh.ClientConfig{HostKeyCallback: ssh.InsecureIgnoreHostKey()})
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

func SetupLogging() {
	logSock := "/tmp/log.sock"
	if filepath.Base(os.Args[0]) != "serve" {
		log.SetFlags(0)
		conn, err := net.Dial("unix", logSock)
		assert(err)
		log.SetOutput(conn)
	} else {
		os.Remove(logSock)
		log.Println("Starting log service ...")
		ln, err := net.Listen("unix", logSock)
		assert(err)
		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					break
				}
				go func(conn net.Conn) {
					scanner := bufio.NewScanner(conn)
					for scanner.Scan() {
						log.Println(scanner.Text())
					}
					assert(scanner.Err())
				}(conn)
			}
		}()
	}
}
