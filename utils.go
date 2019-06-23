package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

type ExecError struct {
	cmd  string
	args []string

	stdout   string
	stderr   string
	exitCode int
}

func (e ExecError) Error() string {
	return fmt.Sprintf(
		"Command (%s %s) failed with exitCode: %d\nstdout: %s\nstderr: %s",
		e.cmd,
		e.args,
		e.exitCode,
		e.stdout,
		e.stderr,
	)
}

// CaptureExec captures stdout/stderr and returns it back to you, error in case there was an error (doesn't
// return errorcode)
func CaptureExec(cmdArg string, args ...string) (stdout string, stderr string, exitCode int) {
	var err error

	cmd := exec.Command(cmdArg, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// return exitError.ExitCode()
			return string(stdoutBuf.Bytes()), string(stderrBuf.Bytes()), exitError.ExitCode()
		}
	}

	// if err != nil {
	// 	return string(stdoutBuf.Bytes()), string(stderrBuf.Bytes()), fmt.Errorf("cmd failed \"%s [%s]\"", cmd, args)
	// }

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())

	return outStr, errStr, 0
}

// CheckExec
func CheckExec(Argcmd string, args ...string) (stdout string, stderr string, err error) {
	cout, cerr, exitCode := CaptureExec(Argcmd, args...)
	if exitCode != 0 {
		return cout, cerr, ExecError{
			cmd:  Argcmd,
			args: args,

			stdout:   cout,
			stderr:   cerr,
			exitCode: exitCode,
		}
	}

	return cout, cerr, nil
}

// SSH error return codes:
// 127 - running a cmd that the host doesnt know
// 130 - ctrl-c
// 255 - failed to connect
const ExitCodeSSHUnknownCommand = 127
const ExitCodeSSHCtrlC = 130
const ExitCodeSSHAuthenticationFailure = 255
