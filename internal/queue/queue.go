package queue

import (
	"sync"

	"github.com/codedninja/skener/pkg/xen"
)

type Job interface {
	Process(*Agent)
}

type Agent struct {
	MITMPort string
	DNSPort  string
	IP       string

	VM *xen.VM
}

type Worker struct {
	agent *Agent

	done             sync.WaitGroup
	readyPool        chan chan Job
	assignedJobQueue chan Job

	quit chan bool
}

type JobQueue struct {
	internalQueue     chan Job
	readyPool         chan chan Job
	workers           []*Worker
	dispatcherStopped sync.WaitGroup
	agentsStopped     sync.WaitGroup
	quit              chan bool
}

func NewJobQueue(agentsConnections []*Agent) *JobQueue {
	workerCount := len(agentsConnections)

	workersStopped := sync.WaitGroup{}
	readyPool := make(chan chan Job, workerCount)
	workers := make([]*Worker, workerCount, workerCount)
	for i := 0; i < workerCount; i++ {
		workers[i] = NewWorker(agentsConnections[i], readyPool, workersStopped)
	}

	return &JobQueue{
		internalQueue:     make(chan Job),
		readyPool:         readyPool,
		workers:           workers,
		dispatcherStopped: sync.WaitGroup{},
		agentsStopped:     workersStopped,
		quit:              make(chan bool),
	}
}

func (q *JobQueue) Start() {
	for i := 0; i < len(q.workers); i++ {
		q.workers[i].Start()
	}
	go q.dispatch()
}

func (q *JobQueue) Stop() {
	q.quit <- true
	q.dispatcherStopped.Wait()
}

func (q *JobQueue) dispatch() {
	q.dispatcherStopped.Add(1)
	for {
		select {
		case job := <-q.internalQueue:
			agentChannel := <-q.readyPool
			agentChannel <- job
		case <-q.quit:
			for i := 0; i < len(q.workers); i++ {
				q.workers[i].Stop()
			}
			q.agentsStopped.Wait()
			q.dispatcherStopped.Done()
			return
		}
	}
}

func (q *JobQueue) Submit(job Job) {
	q.internalQueue <- job
}

func NewWorker(agent *Agent, readyPool chan chan Job, done sync.WaitGroup) *Worker {
	return &Worker{
		agent:            agent,
		done:             done,
		readyPool:        readyPool,
		assignedJobQueue: make(chan Job),
		quit:             make(chan bool),
	}
}

func (a *Worker) Start() {
	go func() {
		a.done.Add(1)
		for {
			a.readyPool <- a.assignedJobQueue
			select {
			case job := <-a.assignedJobQueue:
				job.Process(a.agent)
			case <-a.quit:
				a.done.Done()
				return
			}
		}
	}()
}

func (a *Worker) Stop() {
	a.quit <- true
}
