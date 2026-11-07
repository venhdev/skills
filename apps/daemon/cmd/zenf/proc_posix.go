//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

// setDetachedProcess sets platform-specific detached process attributes on POSIX systems.
func setDetachedProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}
