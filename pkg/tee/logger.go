package tee

import (
	"os"

	log "github.com/sirupsen/logrus"
)

type Logger struct {
	file *os.File
	name string
}

func NewLogger(name string) (*Logger, error) {
	file, err := os.CreateTemp("", name)
	if err != nil {
		return nil, err
	}

	return &Logger{
		file: file,
		name: name,
	}, nil
}

func (l *Logger) Write(p []byte) (n int, err error) {
	log.Println(l.name + " " + string(p))
	return l.file.Write(p)
}

func (l *Logger) Close() error {
	return l.file.Close()
}

func (l *Logger) Path() string {
	return l.file.Name()
}
