//go:build windows
// +build windows

package tee

import (
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func newCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func (t *Tee) kill() error {
	if err := t.cmd.Process.Kill(); err != nil {
		log.Error(err)
	}
	return nil
}
