package worker

import (
	"github.com/codedninja/skener/internal/queue"
	"github.com/codedninja/skener/pkg/xen"
	log "github.com/sirupsen/logrus"
)

type Worker struct {
	config *Config

	xen *xen.Client

	// Pool
	agents []*queue.Agent
	queue  *queue.JobQueue
	scans  map[string]*Scan
}

func Run() error {
	worker := &Worker{
		config: &Config{},
	}

	// Load config
	log.Println("worker: loading config")
	if err := worker.loadConfig(); err != nil {
		return err
	}

	// Connect to XenServer
	log.Println("worker: connecting to xen server")
	if err := worker.connectXenServer(); err != nil {
		return err
	}

	// Get possible Agents (VMs)
	log.Println("worker: finding possible agents")
	if err := worker.findPossibleAgents(); err != nil {
		return err
	}

	// Start queue
	log.Println("worker: starting queue")
	worker.startQueue()
	defer worker.queue.Stop()

	// Start http server
	log.Println("worker: starting http server")
	worker.startServer()

	return nil
}
