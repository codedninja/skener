//go:build linux
// +build linux

package tee

import (
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func newCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func (t *Tee) kill() error {
	// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	if err := syscall.Kill(-t.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		log.Error(err)
	}
	return nil
}
