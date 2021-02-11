// Package ocelot ...
package ocelot

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Setup Code
var (
	staticTestJob = &Job{
		ID:          uuid.New(),
		Interval:    time.Millisecond * 200,
		StagingChan: make(chan *JobInstance, 10), // Assuming Buffer of 10 - Passed from config
	}
)

func TestJob_newInstance(t *testing.T) {
	t.Run("Calling Job.newInstance Produces JobInstance", func(t *testing.T) {

		// Create New Instance
		ji := staticTestJob.newInstance()

		// Check that JobInstance attaches Job
		if !reflect.DeepEqual(&ji.Job, staticTestJob) {
			t.Errorf("Job.newInstance produces JobInstance.Job: %+v, want: %+v", ji.Job, staticTestJob)
			return
		}

		// Check that Instance Not Nil/Default
		if reflect.DeepEqual(ji.InstanceID, uuid.UUID{}) {
			t.Errorf("Job.newInstance produces nil UUID: %+v, want: uuid.New()", ji.InstanceID)
		}

	})
}

func TestJob_sendInstance(t *testing.T) {
	t.Run("Calling Job.sendInstance Once Produces 1 JobInstance", func(t *testing.T) {

		// Minimal Required Object for Test
		j := &Job{
			ID:          staticTestJob.ID,
			StagingChan: staticTestJob.StagingChan,
		}

		// Send ONE JobInstance to the channel
		j.sendInstance()

		if len(j.StagingChan) != 1 {
			t.Errorf("Job.sendInstance() produces %v instances, want: %v", len(j.StagingChan), 1)
			return
		}
	})

	t.Run("Calling Job.sendInstance 10x Produces `Cap(Chan)` JobInstances", func(t *testing.T) {

		// Minimal Required Object for Test
		j := &Job{
			ID:          staticTestJob.ID,
			StagingChan: staticTestJob.StagingChan,
		}

		// Send N JobInstances to the Staging Channel
		for i := 0; i < cap(j.StagingChan)-1; i++ {
			j.sendInstance()
		}

		// Ensure No Blocking &&
		if len(j.StagingChan) != cap(j.StagingChan) {
			t.Errorf("subsequent calls to job.sendInstance() produces %v instances, want %v", len(j.StagingChan), cap(j.StagingChan))
			return
		}
	})
}

func TestJob_flushChannel(t *testing.T) {

	// Shared Context for Test Suite
	var (
		ctx, cancel   = context.WithCancel(context.Background())
		staticTestJob = &Job{
			ID:          uuid.New(),
			Interval:    time.Millisecond * 200,
			StagingChan: make(chan *JobInstance, 10), // Assuming Buffer of 10 - Passed from config
		}
	)

	t.Run("Calling Job.flushChannel() clears job.StagingChan", func(t *testing.T) {

		// Start a timer to allow JobInstances for 250ms
		go func() {
			time.Sleep(time.Millisecond * 250)
			cancel()
		}()

		staticTestJob.startSchedule(ctx)

		// Check Job Created 2 JobInstances in Staging Channel
		if len(staticTestJob.StagingChan) != 2 {
			t.Errorf(
				"Calling Job.startSchedule() for 250ms Produces %v JobInstances, Want: %v",
				len(staticTestJob.StagingChan), 2,
			)
		}

		// Flush the Channel
		staticTestJob.flushChannel()

		// Check Exit
		if len(staticTestJob.StagingChan) != 0 {
			t.Errorf(
				"Job.flushChannel() Produces %v JobInstances, Want: %v",
				len(staticTestJob.StagingChan), 0,
			)
		}

		// Flush the Channel - Subsequent Call; Func. Is idepotent
		staticTestJob.flushChannel()

		// Check Exit
		if len(staticTestJob.StagingChan) != 0 {
			t.Errorf(
				"Secondary run of Job.flushChannel() Produces %v JobInstances, Want: %v",
				len(staticTestJob.StagingChan), 0,
			)
		}

		// Check behavior on closed channel; Close && Flush the Channel
		// TODO: CLOSE:
		// 	-  close(staticTestJob.StagingChan)
		staticTestJob.flushChannel()

		// Check Exit
		if len(staticTestJob.StagingChan) != 0 {
			t.Errorf(
				"Job.flushChannel() on Closed Channel Produces %v JobInstances, Want: %v",
				len(staticTestJob.StagingChan), 0,
			)
		}
	})

}

func TestJob_startSchedule(t *testing.T) {

	var (
		staticTestJob = &Job{
			ID:          uuid.New(),
			Interval:    time.Millisecond * 200,
			StagingChan: make(chan *JobInstance, 10), // Assuming Buffer of 10 - Passed from config
		}
	)

	t.Run("Checking Nil Ticker", func(t *testing.T) {

	})

	t.Run("Calling Job.startSchedule and Halting Immediatley Produces 1 JobInstance", func(t *testing.T) {
		// Create Context for TestSuite && defered Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer staticTestJob.flushChannel()

		// Start a timer to allow JobInstances for 5ms
		go func() {
			time.Sleep(time.Millisecond * 5)
			cancel()
		}()

		staticTestJob.startSchedule(ctx)

		if len(staticTestJob.StagingChan) != 1 {
			t.Errorf(
				"Calling Job.startSchedule & running for 5ms Produces %v JobInstance, Want %v",
				len(staticTestJob.StagingChan), 1,
			)
		}
	})

	t.Run("Calling Job.startSchedule for 250ms Produces 2 JobInstance", func(t *testing.T) {
		// Shared Context...
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		defer staticTestJob.flushChannel()

		// Wait on Cancel - Sending one job on start, one in the following 200ms...
		go func() {
			time.Sleep(time.Millisecond * 250)
			cancel()
		}()

		staticTestJob.startSchedule(ctx)
		if len(staticTestJob.StagingChan) != 2 {
			t.Errorf(
				"Calling Job.startSchedule & running for 250ms Produces %v JobInstance, Want %v",
				len(staticTestJob.StagingChan), 2,
			)
		}
	})

	t.Run("Calling Job.startSchedule fills Channel to Capacity", func(t *testing.T) {
		var (
			ctx, cancel = context.WithCancel(context.Background())
		)

		defer staticTestJob.flushChannel()

		// Wait on Cancel - Sending one job on start, >10 in the following 2500ms...
		// This will block @ Capacity...
		go func() {
			time.Sleep(time.Millisecond * 2500)
			cancel()
		}()

		staticTestJob.startSchedule(ctx)

		if len(staticTestJob.StagingChan) != cap(staticTestJob.StagingChan) {
			t.Errorf(
				"Calling Job.startSchedule & running for 2500ms Produces %v JobInstance, Want %v",
				len(staticTestJob.StagingChan), cap(staticTestJob.StagingChan),
			)
		}
	})

	t.Run("Calling Job.startSchedule for 250ms (and 500ms Interval) Produces 1 JobInstance", func(t *testing.T) {
		// Shared Context...
		var (
			ctx, cancel = context.WithCancel(context.Background())
			j           = &Job{
				ID:          staticTestJob.ID,
				Interval:    time.Millisecond * 500,
				StagingChan: staticTestJob.StagingChan,
			}
		)

		defer j.flushChannel()

		// Wait on Cancel - Sending one job on start, one in the following 200ms...
		go func() {
			time.Sleep(time.Millisecond * 250)
			cancel()
		}()

		j.startSchedule(ctx)
		if len(j.StagingChan) != 1 {
			t.Errorf(
				"Calling Job.startSchedule & running for 250ms (w. 500ms Interval) Produces %v JobInstance, Want %v",
				len(j.StagingChan), 1,
			)
		}
	})
}
