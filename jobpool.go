package ocelot

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// JobPool - Collection of jobs that are currently active on the producer
type JobPool struct {
	jobs  []*Job
	stgCh chan *JobInstanceMsg
	wg    sync.WaitGroup
}

// StopJob - Access the underlying job in JobPool by stoping the ticker
// WARNING: Dummy Implementation - Should Stop Jobs by ID or Path; NOT 1st Object
func (jp *JobPool) StopJob() {
	jp.jobs[0].quitCh <- true
}

// sendInstance - creates new JobInstance and sends to job's staging channel
func (jp *JobPool) sendInstance(j *Job, t time.Time) {

	// Instatiates new JobInstance. If the stgCh is not available for
	// send within timeout interval (default: 100ms) then drop this JobInstance
	select {

	case jp.stgCh <- j.newJobInstance(t):
	case <-time.After(time.Millisecond * 100):
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Debug("JobInstance Timeout - Dropped Job")
	}
}

// startSchedule - Start a Job's Ticker, sending JobInstances on a fixed
// Tdelta. Function sends event from ticker.C -> j.Staging
func (jp *JobPool) startSchedule(j *Job) {

	defer func() {
		// On exit, clean up all resources to prevent leaking, closes
		// `j.stgCh` and stops the underlying ticker
		j.ticker.Stop()
		close(j.stgCh)
		log.WithFields(
			log.Fields{"Job ID": j.ID},
		).Warn("Ticker Stopped - No Longer Producing Job")
		jp.wg.Done()
	}()

	// Send first Job on schedule start
	jp.sendInstance(j, time.Now())

	for {

		select {
		// Ticks until quit signal recieved; use go j.sendInstance w. a timeout rather
		// than w. default case to prevent blocking the kill signal or using a ton of CPU
		case t := <-j.ticker.C:
			go jp.sendInstance(j, t)

		// Do not use a common context here. Each schedule has a unique quit sig b/c
		// each job must be individually cancelable
		case <-j.quitCh:
			log.WithFields(
				log.Fields{"Job ID": j.ID},
			).Warn("JobPool Got Quit Sig")
			return
		}
	}
}
