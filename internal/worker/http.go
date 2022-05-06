package worker

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
)

func (w *Worker) startServer() {
	e := echo.New()

	e.Use(middleware.Recover())

	e.POST("/analyze", w.analyzeMalware)
	e.GET("/status/:id", w.getStatus)
	e.GET("/results/:id", w.getResults)

	e.Logger.Fatal(e.Start(w.config.Address))
}

func (w *Worker) analyzeMalware(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		log.Error(err)
		return err
	}

	timeout, err := time.ParseDuration(c.FormValue("timeout"))
	if err != nil {
		log.Error(err)
		return err
	}

	// Temporary file
	src, err := file.Open()
	if err != nil {
		log.Error(err)
		return err
	}
	defer src.Close()

	// Generate uuid
	id := uuid.New()

	// TODO: Move logs to S3 instead of local disk
	// Create folder
	path := filepath.Join(w.config.LogPath, id.String())

	if err := os.Mkdir(path, 0777); err != nil {
		log.Error(err)
		return err
	}

	filePath := filepath.Join(path, file.Filename)

	// Move file to folder
	dst, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		log.Error(err)
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		log.Error(err)
		return err
	}

	// Create job
	scan := &Scan{
		ID:         id,
		Timeout:    timeout,
		Status:     "queued",
		OutputPath: path,
		Filepath:   filePath,
	}

	w.scans[id.String()] = scan

	go w.queue.Submit(scan)

	return c.JSON(http.StatusOK, scan)
}

func (w *Worker) getStatus(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	// Check if job exists in memory
	job, ok := w.scans[id.String()]
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, job)
}

func (w *Worker) getResults(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	// Check if job exists in memory
	job, ok := w.scans[id.String()]
	if !ok || job.Status != "finished" {
		return c.NoContent(http.StatusNotFound)
	}

	path := filepath.Join(w.config.LogPath, id.String())

	if err := w.zipResults(path); err != nil {
		return err
	}

	return c.File(filepath.Join(path, "results.zip"))
}
