package ocelot

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
)

// JobPool - Collection of Jobs registered on the producer
type JobPool struct {
	jobs   []*Job
	workCh map[string]chan *JobInstance // workCh is a misleading name...
	wg     sync.WaitGroup
}

// StartJob - Exported wrapper around j.startSchedule()
func (jp *JobPool) StartJob(ctx context.Context, j *Job) {
	jp.wg.Add(1)
	go jp.startSchedule(j) // Start Producer - Start Ticks
	go jp.Forward(j)       // Start Job Forwarder - Start Job.StgChananel -> jp.JobChan
	jp.jobs = append(jp.jobs, j)
}

// Forward - Forward all incoming jobInstances and release
// the WaitGroup When j.stgCh has been closed...
func (jp *JobPool) Forward(j *Job) {

	// Close Job's staging channel and decrement the counter
	// on function exit
	defer func() {
		jp.wg.Done()
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Warn("Staging Stopped - No Longer Forwarding Job")
	}()

	// Route to correct channel based on handlerType; cannot close this
	// when just one job closes...
	for ji := range j.stgCh {
		log.WithFields(
			log.Fields{"Job ID": j.ID, "Instance ID": ji.InstanceID},
		).Trace("JobInstance Created")
		jp.workCh[j.params["type"].(string)] <- ji
	}
}

// gatherJobs - Calls Forward JobInstances from individual Job producers'
// channels to the central JobPool Channel
func (jp *JobPool) gatherJobs() {

	jp.wg.Add(len(jp.jobs))
	for _, j := range jp.jobs {
		go jp.Forward(j)
	}
}

// StopJob - Access the underlying job in JobPool by stoping the ticker
// WARNING: A violation of close on producer side rule...
// StopJob close the producer (the ticker), which later causes gatherJobs
// to close the staging channel...
func (jp *JobPool) StopJob() {
	// WARNING: Dummy Implementation - Should Stop Jobs by ID or Path; NOT 1st Object
	// by FIFO...
	jp.jobs[0].quitCh <- true
}
