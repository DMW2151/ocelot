// Package ocelot ...
package ocelot

import (
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Job - Template for a Job
type Job struct {
	// ID - UUID for each job, uniquely resolves to a `Job` and Session
	ID uuid.UUID

	// Job Specific Channel; used to communicate results to a central
	// Producer channel - Attach Ticker && Create New Ticker to Modify
	stgCh chan *JobInstance

	// QuitSig - Used to stop the job's ticker externally
	quitSig chan int

	// Pass any and all params here needed for the worker.
	// WARNING: MUST BE GOB Encodeable!
	Params map[string]interface{} `yaml:"params"`

	// Tdelta between creation of new job instances
	Tdelta time.Duration `yaml:"tdelta"`

	// Unexported ticker; used to schedule job freq.
	ticker *time.Ticker
}

// JobInstance - Instance of a Job with the original template
// attached as metadata
type JobInstance struct {
	InstanceID uuid.UUID // Randomly generated UUID for each instance,
	CTime      time.Time // Instance Create Time
	Success    bool
}

// FromConfig - Generate a new Job from config
func (j *Job) FromConfig(jcb int) {
	// Set UUID for Instance
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}

	// Set Other params; ticker and Tdelta
	j.stgCh = make(chan *JobInstance, jcb)
	j.ticker = time.NewTicker(j.Tdelta)
	j.quitSig = make(chan int, 1)
}

// newJobInstance - Creates new JobInstance using parent Job as a template
func (j *Job) newJobInstance(t time.Time) *JobInstance {
	return &JobInstance{
		InstanceID: uuid.New(),
		CTime:      t,
	}
}

// sendInstance - creates new JobInstance and sends to job's staging channel
func (j *Job) sendInstance(t time.Time) {

	// Set timeout, if the stgCh is not available for
	// send within 100ms (TODO: can be configurable) then drop
	// this JobInstance
	select {
	// Instatiates new JobInstance w. current time
	case j.stgCh <- j.newJobInstance(t):
	case <-time.After(100 * time.Millisecond):
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Debug("JobInstance Timeout")
	}
}

// startSchedule - Start a Job's Ticker, sending JobInstances on a fixed
// Tdelta. Function sends event from ticker.C -> j.Staging
func (jp *JobPool) startSchedule(j *Job) {

	defer func() {
		// On exit, close the channels this function sends on
		// indirectly, this is `j.stgCh` and the underlying
		// ticker
		j.ticker.Stop()
		close(j.stgCh)
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Warn("Ticker Stopped - No Longer Producing Job")
	}()

	// Send first Job on schedule start
	j.sendInstance(time.Now())
	j.ticker = time.NewTicker(j.Tdelta)

	for {
		select {
		// Ticks until quit signal recieved; use go j.sendInstance
		// to prevent blocking the kill signal
		case t := <-j.ticker.C:
			go j.sendInstance(t)

		// Do not use a common context here because each job must be
		// individually cancelable
		case <-j.quitSig:
			log.WithFields(
				log.Fields{"Job ID": j.ID},
			).Warn("JobPool Got Quit Sig")
			return
		}
	}
}
