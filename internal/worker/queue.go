package worker

import (
	"os"
	"path/filepath"
	"time"

	agentClient "github.com/codedninja/skener/pkg/agent"
	log "github.com/sirupsen/logrus"

	"github.com/codedninja/skener/internal/queue"
	"github.com/codedninja/skener/internal/route"
	"github.com/codedninja/skener/internal/tool"
	"github.com/google/uuid"
)

type Scan struct {
	ID         uuid.UUID
	Timeout    time.Duration
	Status     string
	OutputPath string
	Filepath   string

	w *Worker
}

func (s *Scan) Process(agent *queue.Agent) {
	s.Status = "starting"
	client := agentClient.NewClient(agent.IP + ":6060")

	log.Printf("Adding IPTable rules for %s", s.ID)
	routes, err := route.ApplyRules(agent.IP, agent.MITMPort, agent.DNSPort)
	if err != nil {
		log.Error(err)
		return
	}
	defer routes.Delete()

	log.Printf("Starting analysis tools for %s", s.ID)
	tools, err := tool.StartTools("0.0.0.0", agent.MITMPort, agent.DNSPort, s.OutputPath)
	if err != nil {
		log.Error(err)
		return
	}
	// TODO: Handle errors for tools

	s.Status = "uploading"
	log.Printf("uploading file %s for %s", s.Filepath, s.ID)
	if err := client.UploadFile(s.Filepath); err != nil {
		log.Error(err)
		return
	}

	s.Status = "uploaded"
	log.Printf("file %s uploaded for %s", s.Filepath, s.ID)

	s.Status = "analyzing"
	log.Printf("analyzing file %s for %s", s.Filepath, s.ID)
	if err := client.Execute(); err != nil {
		log.Error(err)
		return
	}

	log.Printf("Waiting for %s for %s to finish", s.Timeout, s.ID)
	time.Sleep(s.Timeout)

	s.Status = "terminating"
	log.Printf("Terminating analysis for %s", s.ID)
	logs, err := client.Finish()
	if err != nil {
		log.Error(err)
		return
	}

	s.Status = "logging"
	log.Printf("Writing logs for %s", s.ID)
	f, err := os.Create(filepath.Join(s.OutputPath, "agent.log"))
	if err != nil {
		log.Error(err)
		return
	}
	defer f.Close()

	if _, err := f.Write([]byte(logs)); err != nil {
		log.Error(err)
		return
	}

	log.Printf("Stopping analysis tools for %s", s.ID)
	tools.Stop()

	s.Status = "reverting"
	log.Printf("Reverting VM for %s", agent.IP)
	if err := agent.VM.RevertToLastSnapshot(); err != nil {
		log.Error(err)
	}

	s.Status = "resuming"
	log.Printf("Resuming VM for %s", agent.IP)
	if err := agent.VM.Resume(); err != nil {
		log.Error(err)
	}

	s.Status = "finished"
	log.Printf("Finished analysis for %s", s.ID)

	// TODO: Upload logs to S3
}

func (w *Worker) startQueue() {
	w.queue = queue.NewJobQueue(w.agents)
	w.scans = make(map[string]*Scan)

	w.queue.Start()
}
