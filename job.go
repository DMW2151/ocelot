// Package ocelot ...
package ocelot

import (
	"crypto/sha256"
	"fmt"
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

	// quitCh - Used to stop the job's ticker externally
	quitCh chan bool

	// Unexported ticker; used to schedule job freq.
	ticker *time.Ticker

	// Pass any and all params here needed for the worker.
	// WARNING: MUST BE GOB Encodeable!
	params map[string]interface{}
}

// JobInstance - Instance of a Job
type JobInstance struct {
	InstanceID uuid.UUID // Randomly generated UUID for each instance,
	CTime      time.Time // Instance Create Time
	Success    bool
}

// JobConfig - Read from Yaml...
type JobConfig struct {
	ID          uuid.UUID              `yaml:"id"`
	Sendtimeout time.Duration          `yaml:"send_timeout"`
	Tdelta      time.Duration          `yaml:"tdelta"`
	StgBuffer   int                    `yaml:"stg_buffer"`
	Params      map[string]interface{} `yaml:"params"`
}

// updateConfig - Augment a new Job from config, adds UUID, Stg channel,
// and Ticker...
func yieldJob(jc *JobConfig) *Job {

	// Generate Static Hash for Each Job If UUID is Not Generated
	if jc.ID == uuid.Nil {
		jc.ID = generateStaticUUID(
			[]byte(fmt.Sprintf("%v", jc.Params)),
		)
	}

	// If buffer is negative; set to 0, prevent panic...
	if b := (jc.StgBuffer < 0); b {
		log.Warn("Require Job Stage Buffer Non Neg (got %d), setting to 0", jc.StgBuffer)
		jc.StgBuffer = 0
	}

	return &Job{
		ID:     jc.ID,
		stgCh:  make(chan *JobInstance, 0),
		quitCh: make(chan bool, 1),
		ticker: time.NewTicker(jc.Tdelta),
		params: jc.Params,
	}

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

	// Instatiates new JobInstance w. ctime if the stgCh is not available for
	// send within timeout interval (default: 100ms) then drop this JobInstance
	select {

	case j.stgCh <- j.newJobInstance(t):
	case <-time.After(time.Millisecond * 100):
		// TODO: All
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

	for {
		select {
		// Ticks until quit signal recieved; use go j.sendInstance
		// to prevent blocking the kill signal
		case t := <-j.ticker.C:
			go j.sendInstance(t)

		// Do not use a common context here because each job must be
		// individually cancelable
		case <-j.quitCh:
			log.WithFields(
				log.Fields{"Job ID": j.ID},
			).Warn("JobPool Got Quit Sig")
			return
		}
	}
}

// Using a static Ccelot hash, generate an ID from content
func generateStaticUUID(b []byte) (uid uuid.UUID) {
	encUUID, _ := uuid.Parse("bcf3070f-7898-4399-bcae-4fcce2b451f5")
	uid = uuid.NewHash(sha256.New(), encUUID, b, 3)
	return uid
}
