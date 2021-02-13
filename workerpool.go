// Package ocelot -
package ocelot

import (
	"context"
	"encoding/gob"
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
		t         = time.NewTicker(time.Millisecond * 15000) // TESTING
		sessionID = uuid.New()
	)

	wp.StartWorkers(ctx, sessionID)

	// Send Initial Message To Server ...
	enc := gob.NewEncoder(wp.Connection)
	enc.Encode(&wp.Params.HandlerType)

	// There we go...
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case errChan <- dec.Decode(&j):
				log.Println("Heree...") // WARNING!: WorkerPool is Still Recieving Jobs after Shutdowm...
			}
		}
	}()

	for {
		// The Decoder reads data from server and unmarshals into a Job object
		// values from errChan will be nil
		select {

		case err := <-errChan:
			// Either Server Encoder buffer is off and cannot be recovered,
			// or Client has been closed and buffer is incomplete -> Cancel & Exit
			if (err != nil) && (err != io.EOF) {
				log.Errorf("Failed to Unmarshal Job From Server: %e", err)
				cancel()
				return
			}

			if err == io.EOF { // Server has shutdown (or otherwise recieves no data recieved)
				log.WithFields(
					log.Fields{"Session ID": sessionID},
				).Errorf("No Data Received: %v", err)
				cancel()
				return
			}

			// No Errors -> Send Job to WorkerPool and continue processing data...
			wp.Pending <- j
			log.WithFields(
				log.Fields{
					"Session ID":  sessionID,
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
					"Duration":    -1 * j.CTime.Sub(time.Now()),
				},
			).Debug("WorkerPool Recieved Job")

		// Recieves a Cancelfunc() call; either from the case(s) above or user
		case <-ctx.Done():
			log.WithFields(
				log.Fields{
					"Session ID":    sessionID,
					"Worker Addr":   wp.Local,
					"Producer Addr": wp.Remote,
				},
			).Warn("WorkerPool Shutdown")
			wp.Close()
			return

		// TESTING: Workers recieve remaining jobs, depending on the buffer size,
		// this can take a bit...
		case <-t.C:
			log.WithFields(
				log.Fields{
					"Session ID":    sessionID,
					"Worker Addr":   wp.Local,
					"Producer Addr": wp.Remote,
				},
			).Warn("WorkerPool Timeout")
			cancel()
		}

	}
}

// Close - Wrapper around the net.Conn close function, also
// Closes the Pool's Pending channel
func (wp *WorkerPool) Close() {

	// On close - Release Connection && Pool's Pending Channel
	wp.Connection.Close()
	close(wp.Pending)

	log.WithFields(
		log.Fields{
			"Worker Addr":   wp.Local,
			"Producer Addr": wp.Remote,
		},
	).Warn("Connection Closed")
}

// start - Execute the Worker
func (wp *WorkerPool) start(ctx context.Context, wg *sync.WaitGroup, sessionuuid uuid.UUID) {

	defer wg.Done()

	enc := gob.NewEncoder(wp.Connection)

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
			log.Fields{
				"Job ID":      ji.Job.ID,
				"Instance ID": ji.InstanceID,
				"Success":     ji.Success,
			},
		).Info("Encoding...")

		// Write back to server; sends whenever job is done...
		// Shouldn't encode multiple jobs before read...
		select {
		default:
			enc.Encode(&ji)
		case <-ctx.Done():
			enc = nil
		}

	}
}
