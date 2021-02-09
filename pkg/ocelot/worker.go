// Package ocelot ...
package ocelot

import (
	"sync"

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

	// IP and IP of Producer
	Host string
	Port string
}

// StartWorkers ...
func (wp *WorkerPool) StartWorkers() {

	var wg sync.WaitGroup

	// Start Workers
	log.WithFields(
		log.Fields{"Workers": wp.Params.NWorkers},
	).Info("Started Workers")

	for i := 0; i < wp.Params.NWorkers; i++ {
		wg.Add(1)
		go wp.start(&wg)
	}

	// Wait...
	go func() {
		wg.Wait()
	}()
}

// Execute the Worker
func (wp *WorkerPool) start(wg *sync.WaitGroup) {

	defer wg.Done()

	for j := range wp.Pending {

		err := wp.Params.Func(&j)

		// Report Results to logs
		if err != nil {
			log.WithFields(
				log.Fields{
					"Error":       err,
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
				}).Warn("Job Failed")
		}

		log.WithFields(
			log.Fields{
				"Job ID":      j.Job.ID,
				"Instance ID": j.InstanceID,
			}).Info("Job Success")
	}

}
