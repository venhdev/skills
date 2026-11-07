//go:build windows

package main

import (
	"os/exec"
	"syscall"
)

// setDetachedProcess sets platform-specific detached process attributes on Windows.
func setDetachedProcess(cmd *exec.Cmd) {
	const CREATE_NEW_PROCESS_GROUP = 0x00000200
	const DETACHED_PROCESS = 0x00000008
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
	}
}
