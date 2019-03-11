package main

import (
	"bytes"
	"os/exec"
	"syscall"
)

func CaptureCommand(name string, args ...string) string {
	cmd := exec.Command(name, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		panic(err)
	}

	return out.String()
}

func GetExitCode(err error) int {

	if err == nil {
		return 0
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return -1
	}

	// This works on both Unix and Windows. Although package
	// syscall is generally platform dependent, WaitStatus is
	// defined for both Unix and Windows and in both cases has
	// an ExitStatus() method with the same signature.
	if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
		return status.ExitStatus()
	}

	return -2
}
