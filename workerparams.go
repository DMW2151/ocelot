// Package ocelot ...
package ocelot

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// JobHandler - Interface for Object that processes
// a jobInstance
type JobHandler interface {
	Work(j *JobInstance) error
}

// WorkParams is a factory struct for WorkerPools
type WorkParams struct {
	// Name of Handler...
	HandlerType string

	// Number of goroutines to launch taking work
	NWorkers int

	// Job Channel, buffer to allow on server
	MaxBuffer int

	// Most likely User defined function of type
	// JobHandler that workers in this pool execute
	Handler JobHandler

	// IP and Host of Producer
	Host string
	Port string

	// How long to wait on Producer to respond
	DialTimeout time.Duration
}

// NewWorkerPool - Factory for creating a WorkerPool object
// from a Config
func (wpa *WorkParams) NewWorkerPool() (*WorkerPool, error) {

	// Init Connection
	c, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%s", wpa.Host, wpa.Port),
		wpa.DialTimeout,
	)

	if err != nil {
		// Difficult to Test, but this MUST be a Fatal Exit
		log.WithFields(
			log.Fields{"Error": err},
		).Fatal("Failed to Dial Producer")
	}

	return &WorkerPool{
		Connection: &c,
		Params:     wpa,
		Pending:    make(chan JobInstance, wpa.MaxBuffer),
	}, nil
}
