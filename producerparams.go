package ocelot

import (
	"net"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// ProducerConfig - Factory Struct for Producer
type ProducerConfig struct {
	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int

	// Addresses to Listen for New Connections
	ListenAddr string

	// Max Active Connections to producer
	MaxConnections int
}

// newListener from config...
func (cfg *ProducerConfig) newListener() (net.Listener, error) {

	var l, err = net.Listen("tcp", cfg.ListenAddr)

	// Not Testable...Can't Ensure that Exits with Fail...
	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": cfg.ListenAddr},
		).Error("Failed to Start Producer", err)
		return nil, err
	}

	log.WithFields(
		log.Fields{"Producer Addr": cfg.ListenAddr},
	).Info("Listening")

	return l, nil
}

// NewProducer - Create New Server, Initializes connection in function
func (cfg *ProducerConfig) NewProducer(js []*Job) *Producer {

	l, err := cfg.newListener()

	if err != nil {
		return &Producer{} // Exit
	}

	return &Producer{
		Listener: l,
		JobPool: &JobPool{
			Jobs:    js,
			JobChan: make(chan *JobInstance, cfg.JobChannelBuffer),
			wg:      sync.WaitGroup{},
		},
		OpenConnections:  make([]*net.Conn, cfg.MaxConnections),
		NOpenConnections: 0,
		Config:           cfg,
		Sem:              semaphore.NewWeighted(int64(cfg.MaxConnections)),
	}
}
