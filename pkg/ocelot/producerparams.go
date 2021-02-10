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
func (cfg *ProducerConfig) newListener() net.Listener {

	var l, err = net.Listen("tcp", cfg.ListenAddr)

	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": cfg.ListenAddr},
		).Fatal("Failed to Start Producer", err)
	}

	log.WithFields(
		log.Fields{"Producer Addr": cfg.ListenAddr},
	).Info("Listening")

	return l
}

// NewProducer - Create New Server, Initializes connection in function
func (cfg *ProducerConfig) NewProducer(l net.Listener, js []*Job) *Producer {

	// Combine to create Producer object
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
