// Package ocelot ...
package ocelot

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Job - Gives a Path and an Interval for a specific URL
type Job struct {
	ID         uuid.UUID
	InstanceID uuid.UUID
	Path       string
}

// JobMeta - Server Side Information on Jobs
type JobMeta struct {
	ID            uuid.UUID
	Interval      time.Duration
	BaseInterval  time.Duration
	LastExecTime  time.Time
	BackoffPolicy BackoffPolicy
	Ticker        *time.Ticker
	Path          string
	Sem           *semaphore.Weighted
}

// JobStatus - Gives a Status of each Job Request
type JobStatus struct {
	JobID      uuid.UUID
	InstanceID uuid.UUID
	Success    bool
}

// JobConfig -
type JobConfig struct {
	Path            string
	Interval        time.Duration
	MaxRetries      int
	BackoffFunction BackoffFunc
}

func newJobMeta(jConfig *JobConfig) *JobMeta {
	j := &JobMeta{
		ID:           uuid.New(),
		Interval:     jConfig.Interval,
		BaseInterval: jConfig.Interval,
		LastExecTime: time.Time{},
		BackoffPolicy: BackoffPolicy{
			BackoffFunc: jConfig.BackoffFunction,
			MaxRetries:  jConfig.MaxRetries,
			Retry:       0,
			C:           make(chan bool, 1),
		},
		Ticker: &time.Ticker{},
		Path:   jConfig.Path,
		Sem:    semaphore.NewWeighted(1),
	}

	return j
}

// send - send a single job to the server's job's channel
// instatiate a new UUID for each request
func (j *JobMeta) sendInstance(JobChan chan<- *Job) {
	// Create new Job Instance...
	ji := Job{ID: j.ID, Path: j.Path, InstanceID: uuid.New()}
	// fmt.Println("%+v", ji)
	log.WithFields(log.Fields{"Job ID": ji.ID, "Instance ID": ji.InstanceID}).Info("Created Job")

	JobChan <- &ji

}

// StartJob - Starts a Job's Ticker, sends jobs to the jobs channel
// on a fixed interval
func (j *JobMeta) StartJob(ctx context.Context, JobChan chan<- *Job) {
	// Send on Server Init + Create a ticker to
	j.sendInstance(JobChan)
	j.Ticker = time.NewTicker(j.Interval)

	defer func() {
		j.Ticker.Stop()
	}()

	for {
		select {
		// Interval has passed - Put Job into Jobs Channel
		case <-j.Ticker.C:
			j.sendInstance(JobChan)

			// Delay until Next, No quick runs...
			delay := time.NewTimer(j.Interval - time.Millisecond*5)
			<-delay.C

		case <-j.BackoffPolicy.C:
			// Sent an exit signal based on the job's backoff policy;
			// Do not restart in this session...
			j.Ticker.Stop()
			return

		case <-ctx.Done():
			// Server shutting down...
			return
		}
	}

}
