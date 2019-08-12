package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// Responsible for managing an SSH Control-Master that we require for not having to type the password of the host in everytime

func AssembleDefaultSocketPath(host string) string {
	return fmt.Sprintf("/tmp/sshctls/%s/%s", os.Getenv("USER"), host)
}

// ControlMaster manages a SSHControlMaster process
type ControlMaster struct {
	host       string
	SocketPath string

	Cmd *exec.Cmd
}

// NewMaster returns a constructed ControlMaster instance
func NewMaster(socketPath string, host string) *ControlMaster {
	result := &ControlMaster{
		SocketPath: socketPath,
		host:       host,
	}

	// @TODO: check for stale socket-file without running instance
	if !result.Setup() {
		return nil
	}

	return result
}

func (master *ControlMaster) forkSSH(socketPath string, host string) (errorChan chan error) {
	errorChan = make(chan error, 1)

	master.Cmd = exec.Command(
		"ssh",
		"-M", // master
		"-S", socketPath,
		"-N", // don't execute a remote command, just block
		host,
	)

	master.Cmd.Stdin = os.Stdin
	master.Cmd.Stdout = os.Stderr
	master.Cmd.Stderr = os.Stderr

	err := master.Cmd.Start()
	if err != nil {
		log.Error(err)
	}

	// @XXX poll until socket is created >.<
	for {
		select {
		case err := <-errorChan:
			panic(err)
		default:
			_, err := os.Stat(socketPath)

			if err == nil { // file exists
				goto End
			} else if os.IsNotExist(err) { // file does not exist
				time.Sleep(40 * time.Millisecond)
			} else { // unknown error
				panic(err)
			}
		}
	}

End:
	return
}

// Setup returns false if master was already running, the requires cleanupSocket
func (master *ControlMaster) Setup() bool {
	var err error

	if !FileExists(master.SocketPath) { // spawn ssh master if necessary
		err = os.MkdirAll(filepath.Dir(master.SocketPath), os.ModePerm)
		if err != nil {
			panic(err)
		}

		_ = master.forkSSH(master.SocketPath, master.host)

		return true
	}

	return false
}

func (master *ControlMaster) cleanup() {
	var err error

	if master != nil && master.Cmd != nil {
		// clean up master process
		if master.Cmd.Process != nil {
			err := master.Cmd.Process.Kill()
			if err != nil {
				log.Error(err)
			}
		}

		// clean up stale socket file
		if FileExists(master.SocketPath) {
			err = os.Remove(master.SocketPath)
			if err != nil {
				log.Error(err)
			}
		}
	}
}
