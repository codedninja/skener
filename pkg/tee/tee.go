package tee

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Tee struct {
	cmd    *exec.Cmd
	logger *Logger
}

func Run(name string, args ...string) (*Tee, error) {
	var err error
	tee := &Tee{
		cmd: exec.Command(name, args...),
	}

	tee.logger, err = NewLogger(filepath.Base(name))
	if err != nil {
		return nil, err
	}

	tee.cmd.Stdout = tee.logger
	tee.cmd.Stderr = tee.logger

	if err := tee.cmd.Start(); err != nil {
		return nil, err
	}

	return tee, nil
}

func (t *Tee) Interrupt() error {
	if err := t.cmd.Process.Signal(os.Interrupt); err != nil {
		return err
	}

	if _, err := t.logger.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	return nil
}

func (t *Tee) Close() error {
	if err := t.cmd.Process.Kill(); err != nil {
		return err
	}

	if err := t.logger.Close(); err != nil {
		return err
	}

	return nil
}

func (t *Tee) Path() string {
	return t.logger.Path()
}
