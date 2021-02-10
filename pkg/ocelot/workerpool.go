// Package ocelot -
package ocelot

import (
	"context"
	"encoding/gob"
	"io"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// WorkerPool - net connection with WorkParams attached..
// WorkParams set channel buffering, concurrency, etc. of
// WorkerPool
type WorkerPool struct {
	Connection *net.Conn
	Params     *WorkParams
	Pending    chan JobInstance
}

// StartWorkers ...
func (wp *WorkerPool) StartWorkers() {

	var wg sync.WaitGroup

	// Start a GR for each worker in the pool...
	wg.Add(wp.Params.NWorkers)
	for i := 0; i < wp.Params.NWorkers; i++ {
		go wp.start(&wg)
	}

	log.WithFields(
		log.Fields{
			"Worker Addr":   (*wp.Connection).LocalAddr().String(),
			"Producer Addr": (*wp.Connection).RemoteAddr().String(),
		},
	).Debugf("Started %d Workers", wp.Params.NWorkers)

	// Block Forever...
	go func() {
		wg.Wait()
	}()
}

// AcceptWork - Listens for work coming from server...
func (wp *WorkerPool) AcceptWork(ctx context.Context, cancel context.CancelFunc) {

	// Variables instantiated for each Client-> Server connection...
	var (
		errChan = make(chan error, 10)
		j       JobInstance
		dec     = gob.NewDecoder(*wp.Connection)
		t       = time.NewTicker(time.Millisecond * 10000) // REMOVE
	)

	wp.StartWorkers() // Start Workers...

	// Logic for processing incoming requests...
	// Shutdown involves closing the client side jobs channel
	for {

		// The Decoder reads data from server and unmarsals into a Job object
		// Jobs are sent to workers as available...
		errChan <- dec.Decode(&j)

		select {
		// Recieves from `errChan <- dec.Decode(&j)` above; will recieve nil if
		// Job properly marshalled...
		case err := <-errChan:
			if (err != nil) && (err != io.EOF) {
				// Either Server is sending bad Jobs
				//	- In which case the Encoder Buffer cannot be recovered -> Exit
				// 	- Or Client has been closed, buffer is incomplete -> Exit
				log.WithFields(
					log.Fields{"Err": err},
				).Error("Failed to Unmarshal Job From Server")
				cancel()
				return
			}

			if err == io.EOF {
				// Server has shutdown (or client otherwise recieves no data from the Server?)
				// No more jobs to process...
				log.WithFields(
					log.Fields{
						"Error":         err,
						"Worker Addr":   (*wp.Connection).LocalAddr().String(),
						"Producer Addr": (*wp.Connection).RemoteAddr().String(),
					},
				).Errorf("No Data Received")
				cancel()
				return
			}

			// No Errors -> Send Job to WorkerPool and continue processing data...
			wp.Pending <- j
			log.WithFields(
				log.Fields{
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
				},
			).Debug("Workers Recieved Job")

		// Recieves a Cancelfunc() call; either from the case(s) above or user
		case <-ctx.Done():

			log.WithFields(
				log.Fields{
					"Worker Addr":   (*wp.Connection).LocalAddr().String(),
					"Producer Addr": (*wp.Connection).RemoteAddr().String(),
				},
			).Warn("Worker Pool Shutdown")
			wp.Close()
			return

		// REMOVE: keeping this in for testing at the moment
		case <-t.C:
			log.WithFields(
				log.Fields{
					"Session ID":    1,
					"Worker Addr":   (*wp.Connection).LocalAddr().String(),
					"Producer Addr": (*wp.Connection).RemoteAddr().String(),
				},
			).Warn("Ticker Timeout")
			cancel()

		default:
		}

	}
}

// Close - Wrapper around the net.Conn close function
func (wp *WorkerPool) Close() {

	// On close - Release Connection
	(*wp.Connection).Close()
	log.WithFields(
		log.Fields{
			"Worker Addr":   (*wp.Connection).LocalAddr().String(),
			"Producer Addr": (*wp.Connection).RemoteAddr().String(),
		},
	).Warn("Connection Closed")

	// On close - Release Worker Pool - Not sure  if
	// Overly cautious...
	close(wp.Pending)
	log.WithFields(
		log.Fields{
			"Connection - Local": (*wp.Connection).LocalAddr().String(),
		},
	).Warn("Worker Addr")
}

// Execute the Worker
func (wp *WorkerPool) start(wg *sync.WaitGroup) {

	defer wg.Done()

	for j := range wp.Pending {
		// Do the Work; Call the Function...
		err := wp.Params.Func(&j)

		// Report Results to logs
		if err != nil {
			log.WithFields(
				log.Fields{
					"Error":       err,
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
				},
			).Error("Job Failed")
			break
		}

		// Log success...
		log.WithFields(
			log.Fields{
				"Job ID":      j.Job.ID,
				"Instance ID": j.InstanceID,
				"Duration":    -1 * j.CTime.Sub(time.Now()),
			},
		).Debug("Job Success")
	}
}
