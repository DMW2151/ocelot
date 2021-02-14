// Package ocelot ...
package ocelot

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Globals for Testing; reduce boilerplate below...
func TestJobPool_StartJob(t *testing.T) {

	var (
		staticNewJob = Job{
			ID:     uuid.New(),
			Tdelta: time.Millisecond * 150,
			Path:   "https://www.holland.com/global/tourism.htm",
			stgCh:  make(chan *JobInstance, 2),
		}
		staticJP = &JobPool{
			Jobs: []*Job{},
			//JobChan: make(chan *JobInstance, 10), // Assuming JobChannelBuffer of 10; set in ProducerCfg
			wg: sync.WaitGroup{},
		}
	)

	// Check that len jobs = 1 - Added to Location...
	// Check that schedule started -
	t.Run("Check Started", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(time.Millisecond * 1000)
			cancel()
		}()

		staticJP.StartJob(ctx, &staticNewJob)
		time.Sleep(time.Millisecond * 10)

		if len(staticJP.Jobs) != 1 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(staticJP.Jobs), 1,
			)
		}

		j := staticJP.Jobs[0]

		if len(j.stgCh) == 0 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(j.stgCh), 0,
			)
		}

	})
}

func TestJobPool_StopJob(t *testing.T) {

	var (
		staticNewJob = Job{
			ID:     uuid.New(),
			Tdelta: time.Millisecond * 150,
			Path:   "https://www.holland.com/global/tourism.htm",
			stgCh:  make(chan *JobInstance, 2),
		}
		staticJP = &JobPool{
			Jobs: []*Job{},
			//JobChan: make(chan *JobInstance, 10), // Assuming JobChannelBuffer of 10; set in ProducerCfg
			wg: sync.WaitGroup{},
		}
	)

	t.Run("Check Start && Stop...", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())

		// Start...
		// Check that len jobs = 1 - Added to Location...
		// Check that schedule started -
		go func() {
			time.Sleep(time.Millisecond * 1000)
			cancel()
		}()

		staticJP.StartJob(ctx, &staticNewJob)

		if len(staticJP.Jobs) != 1 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(staticJP.Jobs), 1,
			)
		}

		j := staticJP.Jobs[0]
		staticJP.StopJob()

		if j.ticker != nil {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				j.ticker, nil,
			)
		}

	})
}

func TestJobPool_gatherJobs(t *testing.T) {

	var (
		staticNewJob = Job{
			ID:     uuid.New(),
			Tdelta: time.Millisecond * 150,
			Path:   "https://www.holland.com/global/tourism.htm",
			stgCh:  make(chan *JobInstance, 2),
		}
		staticJP = &JobPool{
			Jobs: []*Job{},
			//JobChan: make(chan *JobInstance, 10), // Assuming JobChannelBuffer of 10; set in ProducerCfg
			wg: sync.WaitGroup{},
		}
	)

	t.Run("Check Gather...", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // NOTE: Not Required for Test, Needed For Go Vet

		// Start Jobs && Wait 1000ms - Ensures this stays alive...
		staticJP.StartJob(ctx, &staticNewJob)
		time.Sleep(time.Millisecond * 10)

		// Two Jobs are Registered to the Slice of Jobs...
		if len(staticJP.Jobs) != 1 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(staticJP.Jobs), 1,
			)
			return
		}

		// Ensure None in the Pool Queue before Gather
		if len(staticJP.JobChan) != 0 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(staticJP.JobChan), 0,
			)
			return
		}
		// Run Gather && send to job staging
		staticJP.gatherJobs()
		time.Sleep(time.Millisecond * 10)

		// Ensure sent their init msg...
		if len(staticJP.JobChan) != 1 {
			t.Errorf(
				"Job.newInstance() = %+v, want %+v",
				len(staticJP.JobChan), 1,
			)
			return
		}

		// Allow Time for Full Buffer to Fill...
		time.Sleep(time.Millisecond * 4000)

		// // Ensure both sent their init msg...
		// if len(staticJP.JobChan) != cap(staticJP.JobChan) {
		// 	t.Errorf(
		// 		"Job.newInstance() = %+v, want %+v",
		// 		len(staticJP.JobChan), cap(staticJP.JobChan),
		// 	)
		// 	return
		// }
	})
}
