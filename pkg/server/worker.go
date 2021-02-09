// Package ocelot ...
package ocelot

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

// Callable -
type Callable func(j Job) error

// WorkParams ...
type WorkParams struct {
	NWorkers   int
	JobChan    chan Job
	StatusChan chan JobStatus
	Func       Callable
}

// StartWorkers ...
func (wp *WorkParams) StartWorkers() {

	var wg sync.WaitGroup

	// Start Workers
	log.WithFields(
		log.Fields{"Workers": wp.NWorkers},
	).Info("Started Workers")

	for i := 0; i < wp.NWorkers; i++ {
		wg.Add(1)
		go wp.start(&wg)
	}

	// Wait...
	go func() {
		wg.Wait()
	}()
}

// Start ...
func (wp *WorkParams) start(wg *sync.WaitGroup) {

	defer wg.Done()

	// Recieve value from channel, small struct, better than copying and dereferencing...
	// - https://tam7t.com/golang-range-and-pointers/
	// - https://stackoverflow.com/questions/49123133/sending-pointers-over-a-channel
	for j := range wp.JobChan {
		// User defines the callable...
		err := wp.Func(j) // TODO, there has to be something less awkward than this...
		if err != nil {
			// Communicate the Failure back to the Server...
			log.WithFields(
				log.Fields{"Error": err, "Job ID": j.ID, "Instance ID": j.InstanceID},
			).Warn("Failed to Complete Job")

			wp.StatusChan <- JobStatus{
				InstanceID: j.InstanceID,
				JobID:      j.ID,
				Success:    false,
			}

		} else {
			// Communicate the Success back to the Server...
			wp.StatusChan <- JobStatus{
				InstanceID: j.InstanceID,
				JobID:      j.ID,
				Success:    true,
			}

		}
	}

}
