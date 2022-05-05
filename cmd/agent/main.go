package main

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"github.com/codedninja/skener/pkg/tee"
	"github.com/labstack/echo/v4"
)

var tempDir string = ""

type Agent struct {
	e *echo.Echo

	malwarePath string
	tee         *tee.Tee
	cmd         *exec.Cmd
}

func main() {
	tempDir = os.TempDir()

	s := &Agent{
		e: echo.New(),
	}

	s.e.POST("/malware", s.uploadMalware)
	s.e.POST("/execute", s.executeMalware)
	s.e.POST("/finish", s.finish)

	s.e.Logger.Fatal(s.e.Start(":6060"))
}

func (s *Agent) uploadMalware(c echo.Context) error {
	filename := c.FormValue("filename")

	file, err := c.FormFile("file")
	if err != nil {
		log.Error(err)
		return err
	}

	f, err := file.Open()
	if err != nil {
		log.Error(err)
		return err
	}
	defer f.Close()

	s.malwarePath = filepath.Join(tempDir, filename)

	dst, err := os.OpenFile(s.malwarePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Error(err)
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, f); err != nil {
		log.Error(err)
		return err
	}

	return c.NoContent(http.StatusAccepted)
}

func (s *Agent) executeMalware(c echo.Context) error {
	var err error
	s.tee, err = tee.Run(s.malwarePath)
	if err != nil {
		log.Println(err)
		return err
	}

	return c.NoContent(http.StatusAccepted)
}

func (s *Agent) finish(c echo.Context) error {
	if err := s.tee.Close(); err != nil {
		log.Error(err)
		return err
	}

	return c.File(s.tee.Path())
}
