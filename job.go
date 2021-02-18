// Package ocelot ...
package ocelot

import (
	"fmt"
	"time"

	utils "github.com/dmw2151/ocelot/internal"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Job - Template for a Job - Lives Exclusively on the Producer
type Job struct {
	// ID - UUID for each job, uniquely resolves to a `Job`, deterministic
	// based on values in the job's config...
	ID uuid.UUID

	// Job Specific Channel; used to signal job interval has elapsed to
	// central producer channel
	stgCh chan *JobInstance

	// quitCh - Used to stop the job's ticker externally
	quitCh chan bool

	// Used to schedule job freq.
	ticker *time.Ticker

	// Pass any and all Params here needed for the worker (e.g. bucket, url, etc.)
	// WARNING: MUST BE ProtoBuf Encodeable!
	params map[string]string
}

// JobConfig - Read from Yaml and used to generate Job objects
type JobConfig struct {
	ID        uuid.UUID         `json:"id"`
	Tdelta    time.Duration     `json:"tdelta_ms"`
	StgBuffer int               `json:"stg_buffer"`
	Params    map[string]string `json:"params"`
}

// JobInstance -
type JobInstance struct {
	JobID      string
	InstanceID string
	Ctime      int64
	Mtime      int64
	Success    bool
	Params     map[string]string
}

// createNewJob - Creates a new Job from config, adds UUID, stg channel,
// quit channel, and Ticker to the params defined in yaml
func createNewJob(jc *JobConfig) *Job {

	// Generate Static Hash for Each Job If UUID is Not Generated
	if jc.ID == uuid.Nil {
		jc.ID = utils.GenerateStaticUUID(
			[]byte(
				fmt.Sprintf("%v%s", jc.Params, fmt.Sprint(jc.Tdelta)),
			))
	}

	// NOTE: No longer need this? Before req. channel > 0, now any non-neg value
	// will suffice. If buffer is negative; set to 0, prevent panic...
	if b := (jc.StgBuffer < 0); b {
		log.Warn("Require Job Channel Buffer Non Neg (got %d), setting to 0", jc.StgBuffer)
		jc.StgBuffer = 0
	}

	// Handle for JSON -> GoLang default of 1 NanoSecond = 1 -> 1 millisecond = 10^9
	return &Job{
		ID:     jc.ID,
		stgCh:  make(chan *JobInstance, jc.StgBuffer),
		quitCh: make(chan bool, 1),
		ticker: time.NewTicker(jc.Tdelta * time.Millisecond), // Convert to milliS from nanoS
		params: jc.Params,
	}

}

// newJobInstance - Creates new JobInstance using parent Job as a template
func (j *Job) newJobInstance(t time.Time) *JobInstance {

	ji := &JobInstance{
		JobID:      j.ID.String(),
		InstanceID: uuid.New().String(),
		Ctime:      t.UnixNano(),
		Mtime:      0,
		Success:    false,
		Params:     j.params,
	}

	return ji
}
