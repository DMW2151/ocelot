package ocelot

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

// JobPool - Collection of Jobs Registered on the producer
type JobPool struct {
	Jobs    []*Job
	JobChan map[string]chan *JobInstance
	wg      sync.WaitGroup
}

// StartJob - Exported wrapper around j.startSchedule() for standardization's sake
func (jp *JobPool) StartJob(ctx context.Context, j *Job) {
	// Start the proucer && Add to Jobs && Forward!!
	go j.startSchedule(ctx)
	jp.Jobs = append(jp.Jobs, j)

	go jp.Forward(j)
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
	j.ticker = nil

	log.WithFields(
		log.Fields{"Job": j.ID},
	).Info("Halting Job")
}

// Forward -
func (jp *JobPool) Forward(j *Job) {

	defer jp.wg.Done()
	// Forward all incoming jobInstances and release
	// the WaitGroup When j.StagingChan has been closed...
	log.WithFields(
		log.Fields{
			"Job":      j.ID,
			"Interval": j.Interval,
		},
	).Info("New Job Registered")

	for ji := range j.StagingChan {
		// Route to correct channel...
		jp.JobChan[ji.Job.Params["type"].(string)] <- ji
	}

}

// gatherJobs - Forward  JobInstances from individual Job producers' channels to
// to the central JobPool Channel
func (jp *JobPool) gatherJobs() {

	// For each Job registered, send these to out; the central channel...
	for _, j := range jp.Jobs {
		jp.wg.Add(1)
		go jp.Forward(j)
	}

	// Defered wait; block forever while jobs > 0
	go func() {
		jp.wg.Wait()
		log.Info("Channel Forwarding Released; No More Jobs on Producer...")
	}()
}
