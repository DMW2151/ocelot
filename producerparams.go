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
	MaxConnections int `yaml:"max_connections"`

	// List of Jobs to Init
	Jobs []*Job `yaml:"jobs"`
}

// newListener from config...
func (cfg *ProducerConfig) newListener() (net.Listener, error) {

	var l, err = net.Listen("tcp", os.Getenv("OCELOT_LISTEN_ADDR"))

	// Not Testable...Can't Ensure that Exits with Fail...
	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": os.Getenv("OCELOT_LISTEN_ADDR")},
		).Error("Failed to Start Producer", err)
		return nil, err
	}

	log.WithFields(
		log.Fields{"Producer Addr": os.Getenv("OCELOT_LISTEN_ADDR")},
	).Info("Listening")

	return l, nil
}

// NewProducer - Create New Server, Initializes connection in function
func (cfg *ProducerConfig) NewProducer() *Producer {

	l, err := cfg.newListener()

	if err != nil {
		return &Producer{} // Exit
	}

	// Create Proucer....
	p := Producer{
		Listener: l,
		JobPool: &JobPool{
			Jobs: cfg.Jobs,
			// *JobInstance, cfg.JobChannelBuffer
			JobChan: make(map[string]chan (*JobInstance)),
			wg:      sync.WaitGroup{},
		},
		OpenConnections:  make([]*net.Conn, cfg.MaxConnections),
		NOpenConnections: 0,
		Config:           cfg,
		Sem:              semaphore.NewWeighted(int64(cfg.MaxConnections)),
	}

	// Init Channels for Each Unique Handler...
	for _, j := range p.JobPool.Jobs {
		handlerType := j.Params["type"].(string)

		// Yuck: Assign new channel for each...
		if _, ok := p.JobPool.JobChan[handlerType]; !ok {
			p.JobPool.JobChan[handlerType] = make(chan *JobInstance, cfg.JobChannelBuffer)
		}

	}

	return &p
}
