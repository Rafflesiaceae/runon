package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

// CaptureExec captures stdout/stderr and returns it back to you, error in case there was an error (doesn't
// return errorcode)
func CaptureExec(cmdStr string, args ...string) (stdout string, stderr string, retErr error) {
	var err error

	cmd := exec.Command(cmdStr, args...)
	// fmt.Printf("\n@\n%v\n@\n", args)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		return string(stdoutBuf.Bytes()), string(stderrBuf.Bytes()), fmt.Errorf("cmd failed \"%s [%s]\"", cmdStr, args)
	}

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())

	return outStr, errStr, nil
}

// SSH error return codes:
// 127 - running a cmd that the host doesnt know
// 130 - ctrl-c
// 255 - failed to connect
const ExitCodeSSHUnknownCommand = 127
const ExitCodeSSHCtrlC = 130
const ExitCodeSSHAuthenticationFailure = 255
