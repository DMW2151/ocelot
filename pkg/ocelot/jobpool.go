package ocelot

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

// JobPool - Collection of Jobs Registered on the producer
type JobPool struct {
	Jobs    []*Job
	JobChan chan *JobInstance
	wg      sync.WaitGroup
}

// StartJob - Exported wrapper around j.startSchedule() for standardization's sake
func (jp *JobPool) StartJob(ctx context.Context, j *Job) {
	go j.startSchedule(ctx)
	jp.Jobs = append(jp.Jobs, j)
}

// StopJob - Access the underlying job in JobPool by stoping the ticker
// WARNING: A violation of close on producer side rule...
// StopJob close the producer (the ticker), which later causes gatherJobs
// to close the staging channel...
func (jp *JobPool) StopJob() {
	// WARNING: Dummy Implementation - Should Stop Jobs by ID or Path; NOT 1st Object
	// by FIFO...

	j := jp.Jobs[0]
	j.ticker.Stop()

	log.WithFields(
		log.Fields{"Job": j.ID},
	).Info("Halting Job")
}

// gatherJobs - Forward  JobInstances from individual Job producers' channels to
// to the central JobPool Channel
func (jp *JobPool) gatherJobs() {

	jp.wg.Add(len(jp.Jobs))
	// For each Job registered, send these to out; the central channel...
	for _, j := range jp.Jobs {

		go func(j *Job) {
			// Forward all incoming jobInstances and release
			// the WaitGroup When j.StagingChan has been closed...

			log.WithFields(
				log.Fields{
					"Job":      j.ID,
					"Interval": j.Interval,
				},
			).Info("New Job Registered")

			for ji := range j.StagingChan {
				jp.JobChan <- ji
			}
			jp.wg.Done()
		}(j)
	}

	// Defered wait; block forever while jobs > 0
	go func() {
		jp.wg.Wait()
		log.Info("Channel Forwarding Released; No More Jobs on Producer...")
	}()
}
