// Package ocelot ...
package ocelot

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Callable - Alias for Group of Functions
type Callable func(j *JobInstance) error

// WorkParams ...
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

// StartWorkers ...
func (wp *WorkerPool) StartWorkers() {

	var wg sync.WaitGroup

	// Start a GR for each worker in the pool...
	for i := 0; i < wp.Params.NWorkers; i++ {
		wg.Add(1)
		go wp.start(&wg)
	}

	log.WithFields(
		log.Fields{
			"Worker Addr":   (*wp.Connection).LocalAddr().String(),
			"Producer Addr": (*wp.Connection).RemoteAddr().String(),
		},
	).Debugf("Started %d Workers", wp.Params.NWorkers)

	go func() {
		wg.Wait()
	}()

}

// Execute the Worker
func (wp *WorkerPool) start(wg *sync.WaitGroup) {

	defer wg.Done()

	for j := range wp.Pending {
		// Do the Work; Call the Function...
		err := wp.Params.Func(&j)

		// Report Results to logs
		if err != nil {
			log.WithFields(
				log.Fields{
					"Error":       err,
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
				},
			).Error("Job Failed")
			break
		}

		// Log success...
		log.WithFields(
			log.Fields{
				"Job ID":      j.Job.ID,
				"Instance ID": j.InstanceID,
			},
		).Debug("Job Success")
	}

}
