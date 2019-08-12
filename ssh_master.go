package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
)

// Responsible for managing an SSH Control-Master that we require so we don't have to type the password of the host in everytime

// AssembleDefaultSocketPath returns default socket-path for a given host
func AssembleDefaultSocketPath(host string) string {
	return fmt.Sprintf("/tmp/sshctls/%s/%s", os.Getenv("USER"), host)
}

// ControlMaster manages a SSHControlMaster process
type ControlMaster struct {
	host       string
	SocketPath string

	Cmd *exec.Cmd
}

// NewControlMaster returns a constructed ControlMaster instance
func NewControlMaster(socketPath string, host string) *ControlMaster {
	var err error

	master := &ControlMaster{
		SocketPath: socketPath,
		host:       host,
	}

	// @TODO: check for stale socket-file without running instance
	if FileExists(master.SocketPath) {
		return nil
	}

	{ // no file exists at socketPath, continue creating the master
		err = os.MkdirAll(filepath.Dir(master.SocketPath), os.ModePerm)
		if err != nil {
			panic(err)
		}

		_ = master.forkSSH(master.SocketPath, master.host)
	}

	return master
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

	log.Debugf("ControlMaster.start: %v", master.Cmd.Args)
	err := master.Cmd.Start()
	if err != nil {
		log.Error(err)
	}

	// @XXX poll until socket is created >.<
	// @TODO is errorChan still necessary?
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

// Cleanup cleans up stale resources of the ControlMaster
func (master *ControlMaster) Cleanup() {
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
