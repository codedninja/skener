package tee

import (
	"os/exec"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type Tee struct {
	cmd    *exec.Cmd
	logger *Logger
}

func Run(outputPath string, name string, args ...string) (*Tee, error) {
	var err error
	tee := &Tee{}

	cmd := exec.Command(name, args...)

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	tee.logger, err = NewLogger(outputPath)
	if err != nil {
		return nil, err
	}

	cmd.Stdout = tee.logger
	cmd.Stderr = tee.logger

	tee.cmd = cmd

	go func() {
		log.Printf("Starting %s\n", name)
		if err := tee.cmd.Run(); err != nil && err.Error() != "signal: killed" {
			log.Printf("Error running command:", err)
		}
	}()

	return tee, nil
}

func (t *Tee) Kill() error {
	// https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773
	if err := syscall.Kill(-t.cmd.Process.Pid, syscall.SIGKILL); err != nil {
		log.Error(err)
	}

	if err := t.logger.Close(); err != nil {
		log.Error(err)
		return err
	}

	return nil
}

func (t *Tee) Path() string {
	return t.logger.Path()
}
