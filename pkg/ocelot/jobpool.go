package ocelot

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

// JobPool - Collection on of Jobs on the producer
// TODO: Add Mutex (or Semaphore) andd convert `Jobs`
// as a map uuid -> *Job iff need to modify
type JobPool struct {
	Jobs    []*Job
	JobChan chan *JobInstance
	wg      sync.WaitGroup
}

// StopJob - Access the underlying job in JobPool and modify the ticker
// TODO: cancellation by ID...
func (jp *JobPool) StopJob() {
	// Close the intermediate channel...
	// TODO: Use Cancel by ID...
	// Violation of close on proucer side rule; how can we make this more elegant...
	// stopjob kills the producer (the ticker), gather kills the staging channel...
	j := jp.Jobs[0]
	j.ticker = nil

	//jp.wg.Add(-1)

	log.WithFields(
		log.Fields{
			"Job": j.ID,
		},
	).Info("Halting Job")
}

// StartJob - Exported wrapper around j.startSchedule()
// for clarity...
func (jp *JobPool) StartJob(ctx context.Context, j *Job) {
	// Wrap to j.startSchedule and add to jp metadata
	j.startSchedule(ctx)
	jp.Jobs = append(jp.Jobs, j)
}

// gatherJobs - Forwards the jobs from individual tasks
// to the central writer...
// https://medium.com/justforfunc/two-ways-of-merging-n-channels-in-go-43c0b57cd1de
// TODO: this can run on WP, not producer...
func (jp *JobPool) gatherJobs() {

	jp.wg.Add(len(jp.Jobs))
	// For each Job registered
	for _, j := range jp.Jobs {

		// NOTE: This is a weird pattern, end up with a potential buffer size
		// much greater than the buffer passed to `JobChannelBuffer` in the
		// config

		// Given a `JobChannelBuffer` of 5, and a `StagingChan` buffer of 10
		// Producer can produce 16 events before worker is online

		// Send these to out; the central channel...
		go func(j *Job) {
			// Log that jobs are being forwarded...
			log.WithFields(
				log.Fields{
					"Job": j.ID,
					"Mem": &jp.JobChan,
				},
			).Info("Channel Registered")

			for ji := range j.StagingChan {
				// NOTE: Log here for Super Debugging
				jp.JobChan <- ji

				log.WithFields(
					log.Fields{"Job": j.ID}).Info("Job to Stg Channel")
			}

			jp.wg.Done()
			// Force 'Close' the ticker created in StartSchedule
			// set to nil to free the thread...
			j.ticker = nil
		}(j)
	}

	// Defered wait; block forever.
	go func() {
		jp.wg.Wait()
		log.Info("Channel Forwarding Released; No More Jobs on Producer...")
	}()
}
