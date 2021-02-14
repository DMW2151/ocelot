package ocelot

import (
	"net"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// ProducerConfig - Factory Struct for Producer
type ProducerConfig struct {
	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int `yaml:"job_channel_buffer"`

	// Addresses to Listen for New Connections
	ListenAddr string

	// Max Active Connections to producer
	MaxConn int `yaml:"max_connections"`

	// List of Jobs to Init
	Jobs []*Job `yaml:"jobs"`
}

// newListener from config...
func newListener() net.Listener {

	var l, err = net.Listen("tcp", os.Getenv("OCELOT_LISTEN_ADDR"))

	// Not Testable...Can't Ensure that Exits with Fail...
	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": os.Getenv("OCELOT_LISTEN_ADDR")},
		).Fatal("Failed to Start Producer", err)
	}
	return l
}

// NewProducer - Create New Server, Initializes connection in function
func (cfg *ProducerConfig) NewProducer() *Producer {

	// Create Proucer....
	p := Producer{
		listener: newListener(),
		jobPool: &JobPool{
			Jobs: cfg.Jobs,
			// *JobInstance, cfg.JobChannelBuffer
			JobChan: make(map[string]chan (*JobInstance), cfg.JobChannelBuffer),
			wg:      sync.WaitGroup{},
		},
		openConnections: make([]*net.Conn, cfg.MaxConn),
		config:          cfg,
		sem:             semaphore.NewWeighted(int64(cfg.MaxConn)),
	}

	// Init Channels for Each Unique Handler...
	for _, j := range p.jobPool.Jobs {
		handlerType := j.Params["type"].(string)

		// Yuck: Assign new channel for each...
		if _, ok := p.jobPool.JobChan[handlerType]; !ok {
			p.jobPool.JobChan[handlerType] = make(chan *JobInstance, cfg.JobChannelBuffer)
		}

	}

	return &p
}
