package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/codedninja/skener/internal/queue"
	"github.com/codedninja/skener/pkg/agent"
	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	logPath = "./logs/"
	timeout = 120
)

var logFiles = []string{
	// "mitm.log",
	// "dnschef.log",
	"agent.log",
}

var mitmPorts = []string{
	"8080",
}

var agentsConnections = []string{
	"127.0.0.1:6060",
}

type Server struct {
	e *echo.Echo

	jobs map[string]*Job

	queue *queue.JobQueue
}

func main() {
	s := &Server{
		e:     echo.New(),
		queue: queue.NewJobQueue(agentsConnections),
		jobs:  make(map[string]*Job),
	}

	s.queue.Start()
	defer s.queue.Stop()

	s.e.Use(middleware.Logger())
	// s.e.Use(middleware.Recover())

	s.e.POST("/analyze", s.analyzeMalware)
	s.e.GET("/status/:id", s.getStatus)
	s.e.GET("/results/:id", s.getResults)

	s.e.Logger.Fatal(s.e.Start(":5050"))
}

func (s *Server) analyzeMalware(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}

	// Temporary file
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Generate uuid
	id := uuid.New()

	// Create folder
	path := filepath.Join(logPath, id.String())

	if err := os.MkdirAll(path, 0700); err != nil {
		log.Error(err)
		return err
	}

	filePath := filepath.Join(path, file.Filename)

	// Move file to folder
	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Create job
	job := Job{
		ID:       id,
		Status:   "queued",
		Filename: filePath,
	}

	s.jobs[id.String()] = &job

	log.Println(s.jobs)

	go s.queue.Submit(&job)

	return c.JSON(http.StatusOK, job)
}

func (s *Server) getStatus(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	// Check if job exists in memory
	job, ok := s.jobs[id.String()]
	if !ok {
		return c.NoContent(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, job)
}

func (s *Server) getResults(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return err
	}

	// Check if job exists in memory
	job, ok := s.jobs[id.String()]
	if !ok || job.Status != "finished" {
		return c.NoContent(http.StatusNotFound)
	}

	path := filepath.Join(logPath, id.String())

	if err := s.zipResults(path); err != nil {
		return err
	}

	return c.File(filepath.Join(path, "results.zip"))
}

func (s *Server) zipResults(path string) error {
	archive, err := os.Create(filepath.Join(path, "results.zip"))
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)

	for _, file := range logFiles {
		f, err := os.Open(filepath.Join(path, file))
		if err != nil {
			return err
		}
		defer f.Close()

		w, err := zipWriter.Create(file)
		if err != nil {
			return err
		}

		if _, err = io.Copy(w, f); err != nil {
			return err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}

	return nil
}

type Job struct {
	ID       uuid.UUID
	Status   string
	Filename string
}

func (j *Job) Process(address string) {
	j.Status = "starting"

	// TODO: Setup IP TABLES
	// TODO: Start mitmproxy and dnschef

	log.Printf("Processing job %s", j.ID)
	client := agent.NewAgent(address)

	log.Printf("Uploading file %s to agent at %s", j.Filename, address)
	j.Status = "uploading"
	if err := client.UploadFile(j.Filename); err != nil {
		log.Error(err)
		return
	}
	j.Status = "uploaded"

	log.Printf("Starting analysis on agent at %s", address)
	j.Status = "analyzing"
	if err := client.Execute(); err != nil {
		log.Error(err)
		return
	}

	log.Printf("Waiting for agent to finish")
	time.Sleep(time.Second * time.Duration(timeout))

	logs, err := client.Finish()
	if err != nil {
		log.Error(err)
		return
	}

	// Write logs to file
	f, err := os.Create(filepath.Join(logPath, j.ID.String(), "agent.log"))
	if err != nil {
		log.Error(err)
		return
	}
	defer f.Close()

	if _, err := f.Write([]byte(logs)); err != nil {
		log.Error(err)
		return
	}

	// TODO: Stop mitmproxy and dnschef

	fmt.Println(logs)

	j.Status = "finished"
}
