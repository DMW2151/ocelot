// Package ocelot ...
package ocelot

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

var (
	// ErrOpenConnections - This error is reported...
	ErrOpenConnections = errors.New("no more connection pool slots")
)

// Producer - Object that Manages the Scheduler/Producer/Server
type Producer struct {
	// JobPool, used to pool tickers from multiple jobs to one
	// see full description of struct's fields in jobpool.go
	pool     *JobPool
	connPool *connPool
	config   *ProducerConfig
	quitCh   chan bool
}

// ProducerConfig - Factory Struct for Producer
type ProducerConfig struct {
	// Addresses to Listen for New Workers - Supports
	// Standard IPV4 addressing conventions e.g.
	// [::]:2151, 127.0.0.1:2151, etc...
	ListenAddr []string `json:"listen_addr"`

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

// Used to manage incoming connections, allows N connections to
// `openConnections` based on the weight given to `sem`
type connPool struct {
	// Semaphore used to manage max connections available to
	sem *semaphore.Weighted
	// Record of open connections
	openConnections []OcelotWorkerClient
}

// NewProducer - Create New Server, Initializes listener and
// job channels
func NewProducer() (*Producer, error) {

	cfgServer := parseConfig(
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
			stgCh: make(chan (*JobInstanceMsg), cfgServer.JobChannelBuffer),
		},
		config: cfgServer,
		connPool: &connPool{
			openConnections: make([]OcelotWorkerClient, cfgServer.MaxConn),
			sem:             semaphore.NewWeighted(int64(cfgServer.MaxConn)),
		},
		quitCh: make(chan bool, 1),
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

// RegisterNewStreamingWorker - Adds a worker machine to the producer's connection
// pool that handles bi-directional stream based work
func (p *Producer) RegisterNewStreamingWorker(addr string) bool {

	var (
		waitCh = make(chan bool)   // Cancellation Channel Both Send & Recv Legs
		ji     = &JobInstanceMsg{} // Dummy JobInstance
	)

	// Check if there are slots available, connect or return error and exit
	c, err := p.handleWorkerAssignment(addr)
	if err != nil {
		// Cannot connect - Exit
		log.WithFields(
			log.Fields{"Err": err},
		).Warn("Worker Not Registered")
	}

	stream, err := c.ExecuteStream(context.Background())
	if err != nil {
		// Cannot open stream - Exit
		log.WithFields(
			log.Fields{"Err": err},
		).Warnf("Failed to Open Stream")
		return false
	}

	// Start one goroutine to recieve content back from the worker, in this
	// case, the job status
	go func() {
		for {
			// While Data && !io.EOF, write to Producer side logs below
			ji, err := stream.Recv()
			if err == io.EOF {
				close(waitCh)
				return
			}

			log.WithFields(
				log.Fields{
					"Success": ji.GetSuccess(),
					"CTime":   time.Unix(ji.GetCtime(), 0),
					"MTime":   time.Unix(ji.GetMtime(), 0),
				},
			).Info("Recv Result")
		}
	}()

	// Start one goroutine to send jobs to the worker
	for {
		select {
		case ji = <-p.pool.stgCh:
			if err := stream.Send(ji); err != nil {
				log.WithFields(
					log.Fields{"Err": err, "Conn": c},
				).Warn("Failed to Send Message - Exiting")
				stream.CloseSend()
				<-waitCh
			}
		}
	}
}

func (p *Producer) handleWorkerAssignment(addr string) (OcelotWorkerClient, error) {

	// Producer Dials a Worker...
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		// Returns some form of Net.Error/GRPC.Error re; dial failure
		// shouldn't be fatal, producer can have many workers...
		return nil, err
	}

	// Instatiates new NewOcelotWorkerClient; this is the meanignful
	// connection...
	c := NewOcelotWorkerClient(conn)

	// Grab next empty slot available on the connPool
	if p.connPool.sem.TryAcquire(1) {
		for i, oc := range p.connPool.openConnections {
			if oc == nil {
				p.connPool.openConnections[i] = c
				return c, nil
			}
		}
	}

	// No Connection Slots Remain - Specific Error
	return nil, ErrOpenConnections
}
