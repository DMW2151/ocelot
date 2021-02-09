// Package ocelot ...
package ocelot

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Job - Template for a Job
// TODO: Consider adding a channel for each job instance; this allows
// us to remove failed goroutines from `StartSchedule`
type Job struct {
	// ID - Randomly generated UUID for each job, uniquely resolves to a
	// `Job` and server session
	ID uuid.UUID

	// Minimum Tick Interval of Producer; expect jobs to execute no
	// more frequently than `Interval`
	Interval time.Duration

	Path string // Path of URL to Call...

	// NOTE: Pre-emptively attaching ticker to a Job; might modify
	// schedules rather than stop && restart??
	Ticker *time.Ticker

	// Job Specific Channel; used to communicate results to a central
	// Producer channel - Attach Ticker && Create New Ticker to Modify
	JobStagingChan chan *JobInstance

	// TODO: Mutex (or Semaphore) here if we ever need to modify
}

// JobInstance - Instance of a Job
type JobInstance struct {
	// Randomly generated UUID for each instance, uniquely created
	// for each tick
	InstanceID uuid.UUID

	CTime int64 // Instance Ctime, MTime
	MTime int64
	Job   Job
}

// JobPool - Collection on of Jobs on the producer
type JobPool struct {
	Jobs    []*Job // TODO - as a map iff need to modify
	JobChan chan *JobInstance
	// TODO: Mutex (or Semaphore) here if we ever need to modify
}

// newInstance - creates new instance of JobInstance
// from Job as template
func (j *Job) newInstance() (*JobInstance, error) {
	return &JobInstance{
		Job:        *j,
		InstanceID: uuid.New(),
		CTime:      time.Now().Unix(),
	}, nil
}

// sendInstance - creates new instance of JobInstance
// from Job as template and sends to a jobs channel
func (j *Job) sendInstance(JobChan chan<- *JobInstance) {

	if ji, err := j.newInstance(); err != nil {
		log.Warn("Failed to Create Job Instance")
	} else {
		// Successful Instance Creation
		log.WithFields(
			log.Fields{
				"Job ID":      ji.Job.ID,
				"Instance ID": ji.InstanceID,
			}).Debug("Created Job")

		// Send over channel; will be consumed by an encoder
		// before being send on network

		// Send job instance to Intermediate Channel
		JobChan <- ji
	}
}

// StartSchedule - Start a Job's Ticker, sending jobs to a jobs channel
// on a fixed interval
func (j *Job) StartSchedule(ctx context.Context, JobChan chan<- *JobInstance) {

	// Send first Job on server start; block subsequent sends with time.Ticker()
	// set to interval...
	j.sendInstance(JobChan)
	j.Ticker = time.NewTicker(j.Interval)

	// On exit of StartSchedule; release the ticker
	defer func() {
		j.Ticker.Stop()
		log.WithFields(log.Fields{"Job ID": j.ID}).Info("No Longer Producing Job")
	}()

	// Block While Producer service is up
	for {
		select {
		// Interval has passed - Put Job into Jobs Channel
		case <-j.Ticker.C:
			// NOTE: There is the risk of accumulating tasks here, esp. on
			// server start.
			j.sendInstance(JobChan)
		case <-ctx.Done():
			return

		default: // Ensure `Select` does not block

		}
	}
}

// StopJob - Access the underlying job in JobPool
// and modify the ticker
func (jp *JobPool) StopJob() {
	// Not Implemented
}

// gatherJobs -
func (jp *JobPool) gatherJobs() {
	// Not Implemented
}
