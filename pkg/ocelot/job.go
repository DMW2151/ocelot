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

	// Job Specific Channel; used to communicate results to a central
	// Producer channel - Attach Ticker && Create New Ticker to Modify
	StagingChan chan *JobInstance

	// Try unexported field??
	ticker *time.Ticker
}

// JobInstance - Instance of a Job
type JobInstance struct {
	// Randomly generated UUID for each instance, uniquely created
	// for each tick
	InstanceID uuid.UUID
	CTime      time.Time // Instance Ctime, MTime
	Job        Job
}

// newInstance - creates new instance of JobInstance
// from Job as template
func (j *Job) newInstance() (*JobInstance, error) {
	return &JobInstance{
		Job:        *j,
		InstanceID: uuid.New(),
		CTime:      time.Now(),
	}, nil
}

// sendInstance - creates new instance of JobInstance
// from Job as template and sends to a jobs channel
func (j *Job) sendInstance() {

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

		// Send job instance to Intermediate Channel - It's own staging channel...
		j.StagingChan <- ji
	}
}

// StartSchedule - Start a Job's Ticker, sending jobs to a jobs channel
// on a fixed interval
func (j *Job) startSchedule(ctx context.Context) {

	// Send first Job on server start; block subsequent sends with time.Ticker()
	// set to interval...
	j.sendInstance()

	j.ticker = time.NewTicker(j.Interval)

	// On exit of StartSchedule; release the ticker
	defer func() {
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Info("No Longer Producing Job")
	}()

	// NOTE: There is the risk of accumulating tasks
	// here, esp. on server start.
	for {
		select {
		case <-j.ticker.C:
			if j.ticker != nil {
				j.sendInstance()
			} else {
				return
			}
		case <-ctx.Done():
			return

		default: // No Block!

		}
	}
}
