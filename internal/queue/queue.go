package queue

import "sync"

type Job interface {
	Process(string)
}

// Worker
type Agent struct {
	address string

	done             sync.WaitGroup
	readyPool        chan chan Job
	assignedJobQueue chan Job

	quit chan bool
}

type JobQueue struct {
	internalQueue     chan Job
	readyPool         chan chan Job
	agents            []*Agent
	dispatcherStopped sync.WaitGroup
	agentsStopped     sync.WaitGroup
	quit              chan bool
}

func NewJobQueue(agentsConnections []string) *JobQueue {
	agentCount := len(agentsConnections)

	agentsStopped := sync.WaitGroup{}
	readyPool := make(chan chan Job, agentCount)
	agents := make([]*Agent, agentCount, agentCount)
	for i := 0; i < agentCount; i++ {
		agents[i] = NewAgent(agentsConnections[i], readyPool, agentsStopped)
	}

	return &JobQueue{
		internalQueue:     make(chan Job),
		readyPool:         readyPool,
		agents:            agents,
		dispatcherStopped: sync.WaitGroup{},
		agentsStopped:     agentsStopped,
		quit:              make(chan bool),
	}
}

func (q *JobQueue) Start() {
	for i := 0; i < len(q.agents); i++ {
		q.agents[i].Start()
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
			for i := 0; i < len(q.agents); i++ {
				q.agents[i].Stop()
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

func NewAgent(address string, readyPool chan chan Job, done sync.WaitGroup) *Agent {
	return &Agent{
		address:          address,
		done:             done,
		readyPool:        readyPool,
		assignedJobQueue: make(chan Job),
		quit:             make(chan bool),
	}
}

func (a *Agent) Start() {
	go func() {
		a.done.Add(1)
		for {
			a.readyPool <- a.assignedJobQueue
			select {
			case job := <-a.assignedJobQueue:
				job.Process(a.address)
			case <-a.quit:
				a.done.Done()
				return
			}
		}
	}()
}

func (a *Agent) Stop() {
	a.quit <- true
}
