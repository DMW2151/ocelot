// Package ocelot ...
package ocelot

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Job - Template for a Job
type Job struct {
	// ID - Randomly generated UUID for each job, uniquely resolves
	// to a `Job` and server session
	ID uuid.UUID

	// Minimum Tick Interval of Producer; expect jobs
	// to execute no more frequently than `Interval`
	Interval time.Duration `yaml:"interval"`

	Path string `yaml:"path"` // Path of URL to Call...

	// Job Specific Channel; used to communicate results to a central
	// Producer channel - Attach Ticker && Create New Ticker to Modify
	StagingChan chan *JobInstance

	// Unexported ticker; used to schedule job freq.
	ticker *time.Ticker

	// Pass any and all Params here needed to augment the Path
	// WARNING; MUST BE GOB Encodeable!
	Params map[string]interface{} `yaml:"params"`
}

// JobInstance - Instance of a Job + Template Attached as `Job` field
// for metadata
type JobInstance struct {
	InstanceID uuid.UUID // Randomly generated UUID for each instance,
	CTime      time.Time // Instance Create Time
	Job        Job
}

// FromConfig -
func (j *Job) FromConfig() {
	// Set UUID
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}

	// Set Other
	j.StagingChan = make(chan *JobInstance)
	j.ticker = time.NewTicker(j.Interval)
}

// newInstance - Creates new instance of JobInstance using `Job` as template
func (j *Job) newInstance() *JobInstance {
	return &JobInstance{
		Job:        *j,
		InstanceID: uuid.New(),
		CTime:      time.Now(),
	}
}

// sendInstance - creates new instance of JobInstance and sends to a
// job's staging channel
func (j *Job) sendInstance() {

	ji := j.newInstance()

	// Log successful Instance Creation
	// Send job instance to Intermediate Channel, will be
	// consumed by an encoder before being send on network
	j.StagingChan <- ji

	log.WithFields(
		log.Fields{
			"Job ID":      ji.Job.ID,
			"Instance ID": ji.InstanceID,
		},
	).Debug("Created Job")

}

// StartSchedule - Start a Job's Ticker, sending jobs to a jobs
// channel on a fixed interval
func (j *Job) startSchedule(ctx context.Context) {

	defer func() {
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Info("No Longer Producing Job")
	}()

	// Send first Job on server start + jitter to handle for batch (??)
	// block subsequent sends with time.Ticker()
	j.sendInstance()
	j.ticker = time.NewTicker(j.Interval)

	for {
		select {
		// NOTE: 2021-02-10; Remove second check for && j.Ticker != nil
		case <-j.ticker.C:
			j.sendInstance()
		case <-ctx.Done():
			return
		default:
			if j.ticker == nil {
				return
			}
		}
	}
}

// flushChannel - Helper function for test teardown.
func (j *Job) flushChannel() {
	for {
		select {
		case <-j.StagingChan:
		default:
			return
		}
	}
}

// newMock - Helper function for mocking many jobs
func newMock() *Job {
	return &Job{
		ID:          uuid.New(),
		Interval:    time.Millisecond * 3000,
		Path:        "https://hello.com/en/index.html",
		StagingChan: make(chan *JobInstance, 0),
	}
}
