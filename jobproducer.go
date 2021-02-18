// Package ocelot ...
package ocelot

import (
	"os"
	"sync"
	"time"

	utils "github.com/dmw2151/ocelot/internal"
)

// Producer - Object that Manages the Scheduler/Producer/Server
type Producer struct {
	// JobPool, used to pool tickers from multiple jobs to one
	// see full description of struct's fields in jobpool.go
	pool    *JobPool
	config  *ProducerConfig
	quitCh  chan bool
	MaxConn int `json:"max_connections"`
}

// ProducerConfig - Factory Struct for Producer
type ProducerConfig struct {
	// Addresses to Listen for New Workers - Supports
	// Standard IPV4 addressing conventions e.g.
	// [::]:2151, 127.0.0.1:2151, etc...
	WorkerAddr []string `json:"worker_addr"`

	// Max Active Connections to producer
	MaxConn int `json:"max_connections"`

	// List of Jobs to Init on Server Start
	JobCfgs []*JobConfig `json:"jobs"`

	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int `json:"job_channel_buffer"`

	// Default Timeout for Sends to Staging Channnel
	JobTimeout time.Duration `json:"job_timeout"`

	// Buffer for the pool of multiple jobs sharing one handler...
	HandlerBuffer int `json:"handler_buffer"`
}

// NewProducer - Create New Server, Initializes listener and
// job channels
func NewProducer() (*Producer, error) {

	cfgServer := utils.ParseConfig(
		os.Getenv("OCELOT_PRODUCER_CFG"),
		&ProducerConfig{},
	).(*ProducerConfig)

	// Create Jobs from Config
	jobs := make([]*Job, len(cfgServer.JobCfgs))

	for i, jcfg := range cfgServer.JobCfgs {
		jobs[i] = createNewJob(jcfg)
	}

	// Create Producer, filling in values required from config
	p := Producer{
		pool: &JobPool{
			jobs:  jobs,
			wg:    sync.WaitGroup{},
			stgCh: make(chan (*JobInstance), cfgServer.JobChannelBuffer),
		},
		config:  cfgServer,
		MaxConn: cfgServer.MaxConn,
		quitCh:  make(chan bool, 1),
	}

	return &p, nil
}

// StartJobs - Calls Forward JobInstances from individual Job producers'
// channels to the central JobPool Channel. Called on server start.
// TODO: What do subsequent calls to p.StartJobs do?
func (p *Producer) StartJobs() {

	p.pool.wg.Add(len(p.pool.jobs))

	for _, j := range p.pool.jobs {
		go p.pool.startSchedule(j)
	}

	go func() {
		p.pool.wg.Wait()
	}()
}

// StartJob -  public API wrapper around jp.startSchedule(), replicates
// behavior of p.StartJobs() for a new, user defined job. Does not need to
// be called with server start
func (p *Producer) StartJob(j *Job) {
	p.pool.wg.Add(1)
	go p.pool.startSchedule(j) // Start Producer - Start Ticks
	p.pool.jobs = append(p.pool.jobs, j)
}
