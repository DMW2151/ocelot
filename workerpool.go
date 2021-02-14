// Package ocelot -
package ocelot

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// WorkerPool - net.Conn with WorkParams attached. WorkParams
// set channel buffering, concurrency, etc. of WorkerPool
type WorkerPool struct {
	Connection net.Conn
	Local      string
	Remote     string
	Params     *WorkParams
	Pending    chan JobInstance
}

// StartWorkers - Start Workers, runs `wp.Params.NWorkers`
func (wp *WorkerPool) StartWorkers(ctx context.Context, sessionuuid uuid.UUID) {

	var wg sync.WaitGroup

	wg.Add(wp.Params.NWorkers)
	for i := 0; i < wp.Params.NWorkers; i++ {
		go wp.start(ctx, &wg, sessionuuid)
	}

	log.WithFields(
		log.Fields{"Session ID": sessionuuid},
	).Debugf("Started %d Workers", wp.Params.NWorkers)

	go func() { wg.Wait() }()
}

// AcceptWork - Listens for work coming from server...
func (wp *WorkerPool) AcceptWork(ctx context.Context, cancel context.CancelFunc) {

	var (
		errChan   = make(chan error, 10)
		j         JobInstance
		dec       = gob.NewDecoder(wp.Connection)
		t         = time.NewTicker(time.Millisecond * 10000) // TESTING
		sessionID = uuid.New()
	)

	wp.StartWorkers(ctx, sessionID)

	// Send Initial Message To Server ...
	enc := gob.NewEncoder(wp.Connection)
	enc.Encode(&wp.Params.HandlerType)

	go func() {
		// Constantly Write Work into the Pool, marshal into J
		for {
			err := dec.Decode(&j)
			if err != nil { // If Error Is Not Nil; Check for Op Error
				switch t := err.(type) {

				case *net.OpError:
					if t.Op == "read" {
						log.Errorf("Halt on Op Error: %v", err)
						dec = nil
						close(errChan)
						return
					}
				}
			}

			if err == io.EOF {
				continue
			}

			log.WithFields(
				log.Fields{
					"Instance ID": j.InstanceID,
					"Job":         fmt.Sprintf("%+v", j),
				},
			).Debugf("Worker Read Job: %v", err)
			errChan <- err
		}
	}()

	for {
		// The Decoder reads data from server and marshals into a Job object
		// values from errChan will be nil

		select {

		case err := <-errChan:
			// Either Server Encoder buffer is off and cannot be recovered,
			// or Client has been closed and buffer is incomplete -> Cancel & Exit
			if (err != nil) && (err != io.EOF) {
				log.Errorf("Failed to Unmarshal Job From Server: %v", err)
				cancel()
				return
			}

			// Server has shutdown (or otherwise sent no data)
			// if err == io.EOF {
			// 	log.WithFields(
			// 		log.Fields{"Session ID": sessionID},
			// 	).Errorf("Connection Closed Via Server: %v, %v", err, j)
			// 	cancel()
			// 	return
			// }
			// No Errors -> Send Job to WorkerPool
			wp.Pending <- j

		// Recieves a Cancelfunc() call; begin Workerpool Exit
		case <-ctx.Done():
			log.WithFields(
				log.Fields{"Session ID": sessionID},
			).Warn("WorkerPool Shutdown")
			return

		// TESTING: Workers recieve remaining jobs, depending on the buffer size,
		// this can take a bit...
		case <-t.C:
			log.WithFields(
				log.Fields{"Session ID": sessionID},
			).Warn("WorkerPool Timeout")

			canaryUUID, _ := uuid.Parse("d5b0e6d3-523b-46cc-ad39-8a345acea4cb")
			var canary = &JobInstance{
				InstanceID: canaryUUID,
			}

			enc.Encode(canary)

			cancel()
			//close(errChan)
			wp.Close()

		}

	}
}

// start - Execute the Worker
func (wp *WorkerPool) start(ctx context.Context, wg *sync.WaitGroup, sessionuuid uuid.UUID) {

	defer wg.Done()
	//enc := gob.NewEncoder(wp.Connection)

	for ji := range wp.Pending {
		// Do the Work; Call the Function...
		err := wp.Params.Handler.Work(&ji)

		// Report Results to logs
		if err != nil {
			ji.Success = false
		} else {
			ji.Success = true
		}

		log.WithFields(
			log.Fields{"Instance ID": ji.InstanceID},
		).Info("WorkerPool Finished Job")

		// Write back to server; sends whenever job is done...
		// Shouldn't encode multiple jobs before read...
		//enc.Encode(&ji)
	}
}

// Close - Wrapper around the net.Conn close function, also
// Closes the Pool's Pending channel
// On close - Release Connection && Pool's Pending Channel
func (wp *WorkerPool) Close() {
	log.WithFields(
		log.Fields{
			"Remote": wp.Connection.RemoteAddr().String(),
			"Local":  wp.Connection.LocalAddr().String(),
		}).Warn("Connection Closed...")

	wp.Connection.Close()
	close(wp.Pending)
}
