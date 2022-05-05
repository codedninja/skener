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
	"github.com/codedninja/skener/internal/route"
	"github.com/codedninja/skener/internal/tools"
	"github.com/codedninja/skener/pkg/agent"
	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	logPath = "./logs/"
	timeout = 30
)

var logFiles = []string{
	"agent.log",
	"mitmproxy-stream.log", // Stream file from mitmproxy
	"mitmproxy.log",        // Log from stdout & stderr from mitmproxy
}

var agents = []*queue.Agent{
	{
		IP:          "192.168.86.80",
		Port:        "6060",
		MITMPort:    "8000",
		DNSChefPort: "7000",
	},
}

type Server struct {
	e *echo.Echo

	jobs map[string]*Job

	queue *queue.JobQueue
}

func main() {
	s := &Server{
		e:     echo.New(),
		queue: queue.NewJobQueue(agents),
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

	// Create folder
	path := filepath.Join(logPath, id.String())

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

func (j *Job) Process(a *queue.Agent) {
	outputPath := filepath.Join(logPath, j.ID.String())

	j.Status = "starting"

	log.Printf("Adding IPTable rules")

	address := a.IP + ":" + a.Port

	client := agent.NewAgent(address)
	// Setup IP TABLES
	routeHttp, err := route.NewRoute(a.MITMPort, a.IP, "80")
	if err != nil {
		log.Error(err)
		return
	}

	routeHttps, err := route.NewRoute(a.MITMPort, a.IP, "443")
	if err != nil {
		log.Error(err)
		return
	}

	if err := routeHttp.Apply(); err != nil {
		log.Error(err)
		return
	}
	defer routeHttp.Delete()

	if err := routeHttps.Apply(); err != nil {
		log.Error(err)
		return
	}
	defer routeHttps.Delete()

	// TODO: Start mitmproxy and dnschef
	t, err := tools.StartTools("0.0.0.0", a.MITMPort, outputPath)
	if err != nil {
		log.Error(err)
		return
	}

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
	f, err := os.Create(filepath.Join(outputPath, "agent.log"))
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
	log.Println(t)
	t.Stop()

	fmt.Println(logs)

	j.Status = "finished"
}
