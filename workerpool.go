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
	Connection *net.Conn
	Params     *WorkParams
	Pending    chan JobInstance
}

// StartWorkers - Start Workers, runs `wp.Params.NWorkers`
func (wp *WorkerPool) StartWorkers(sessionuuid uuid.UUID) {

	var wg sync.WaitGroup

	wg.Add(wp.Params.NWorkers)
	for i := 0; i < wp.Params.NWorkers; i++ {
		go wp.start(&wg, sessionuuid)
	}

	log.WithFields(
		log.Fields{
			"Session ID":    sessionuuid,
			"Worker Addr":   (*wp.Connection).LocalAddr().String(),
			"Producer Addr": (*wp.Connection).RemoteAddr().String(),
		},
	).Debugf("Started %d Workers", wp.Params.NWorkers)

	go func() { wg.Wait() }()
}

// AcceptWork - Listens for work coming from server...
func (wp *WorkerPool) AcceptWork(ctx context.Context, cancel context.CancelFunc) {

	var (
		errChan   = make(chan error, 10)
		j         JobInstance
		dec       = gob.NewDecoder(*wp.Connection)
		t         = time.NewTicker(time.Millisecond * 60000) // TESTING
		sessionID = uuid.New()
	)

	wp.StartWorkers(sessionID)

	for {
		// The Decoder reads data from server and unmarsals into a Job object
		// values from errChan will be nil
		errChan <- dec.Decode(&j)
		select {

		case err := <-errChan:
			if (err != nil) && (err != io.EOF) {
				// Either Server Encoder buffer is off and cannot be recovered,
				// or Client has been closed and buffer is incomplete -> Cancel & Exit
				log.WithFields(
					log.Fields{"Err": err},
				).Error("Failed to Unmarshal Job From Server")
				cancel()
				return
			}

			if err == io.EOF {
				// Server has shutdown (or otherwise recieves no data recieved)
				log.WithFields(
					log.Fields{
						"Error":         err,
						"Session ID":    sessionID,
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
					"Worker Addr":   (*wp.Connection).LocalAddr().String(),
					"Producer Addr": (*wp.Connection).RemoteAddr().String(),
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
					"Worker Addr":   (*wp.Connection).LocalAddr().String(),
					"Producer Addr": (*wp.Connection).RemoteAddr().String(),
				},
			).Warn("WorkerPool Timeout")
			cancel()
		default:
		}

	}
}

// Close - Wrapper around the net.Conn close function, also
// Closes the Pool's Pending channel
func (wp *WorkerPool) Close() {

	// On close - Release Connection && Pool's Pending Channel
	(*wp.Connection).Close()
	close(wp.Pending)

	log.WithFields(
		log.Fields{
			"Worker Addr":   (*wp.Connection).LocalAddr().String(),
			"Producer Addr": (*wp.Connection).RemoteAddr().String(),
		},
	).Warn("Connection Closed")
}

// start - Execute the Worker
func (wp *WorkerPool) start(wg *sync.WaitGroup, sessionuuid uuid.UUID) {

	defer wg.Done()

	for j := range wp.Pending {
		// Do the Work; Call the Function...
		err := wp.Params.Handler.Work(&j)

		// Report Results to logs
		if err != nil {
			log.WithFields(
				log.Fields{
					"Session ID":  sessionuuid,
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
				"Session ID":  sessionuuid,
				"Job ID":      j.Job.ID,
				"Instance ID": j.InstanceID,
				"Duration":    -1 * j.CTime.Sub(time.Now()),
			},
		).Info("Job Success")
	}
}