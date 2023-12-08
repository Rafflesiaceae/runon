package main

import (
	"bytes"
	"fmt"
	"os"
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

// CaptureExec captures stdout/stderr and returns it back to you, error in case
// there was an error (doesn't return errorcode)
func CaptureExec(cmdArg string, args ...string) (stdout string, stderr string, exitCode int) {
	var err error

	cmd := exec.Command(cmdArg, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return string(stdoutBuf.Bytes()), string(stderrBuf.Bytes()), exitError.ExitCode()
		}
	}

	outStr, errStr := string(stdoutBuf.Bytes()), string(stderrBuf.Bytes())

	return outStr, errStr, 0
}

func CheckExec(Argcmd string, args ...string) (stdout string, stderr string, err *ExecError) {
	cout, cerr, exitCode := CaptureExec(Argcmd, args...)
	if exitCode != 0 {
		return cout, cerr, &ExecError{
			cmd:  Argcmd,
			args: args,

			stdout:   cout,
			stderr:   cerr,
			exitCode: exitCode,
		}
	}

	return cout, cerr, nil
}

func FileExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
