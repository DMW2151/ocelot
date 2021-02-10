// Package ocelot ...
package ocelot

import (
	"fmt"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
)

// Callable - Alias for Group of Functions
type Callable func(j *JobInstance) error

// WorkParams is a factory struct for WorkerPools
type WorkParams struct {
	// Number of goroutines to launch taking work
	NWorkers int

	// Job Channel, buffer to allow on server
	MaxBuffer int

	// Most likely User defined function of type
	// Callable that workers in this pool execute
	Func Callable

	// IP and Host of Producer
	Host string
	Port string

	// How long to wait on Producer to respond
	DialTimeout time.Duration
}

// NewWorkerPool - Factory function for creating a WorkerPool
func (wpa *WorkParams) NewWorkerPool() (*WorkerPool, error) {

	// Initialize Connection
	// See env vars @ os.Getenv("OCELOT_HOST"), os.Getenv("OCELOT_PORT")
	// TODO: Add to config...
	c, err := net.DialTimeout(
		"tcp",
		fmt.Sprintf("%s:%s", wpa.Host, wpa.Port),
		wpa.DialTimeout,
	)

	// Check Initial Dial ... (Almost?) Always TCP Dial Error (
	// Host is down or wrong) OK to exit w. Code 1 here; no resources
	// created yet...
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Fatal("Failed to Dial Producer")
	}

	// Return WorkerPool Object from Config
	return &WorkerPool{
		Connection: &c,
		Params:     wpa,
		Pending:    make(chan JobInstance, wpa.MaxBuffer),
	}, nil
}
