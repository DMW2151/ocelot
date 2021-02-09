// Package ocelot -
package ocelot

import (
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// WorkerPool - Client Used to Connect a Process to Ocelot Server
type WorkerPool struct {
	Connection *net.Conn
	Params     WorkParams
}

// NewWorkerPool - Factory Function for Client Object
// TODO: Create a Config such that these params can be
func NewWorkerPool() (*WorkerPool, error) {

	jc := make(chan Job, 20)       // Job Channel, From Server
	sc := make(chan JobStatus, 20) // Job Status Channel, to Server

	// Init Params for Worker
	wpParams := WorkParams{
		NWorkers:   10,
		JobChan:    jc,
		StatusChan: sc,
		Func:       HTTPRandomSuccess,
	}

	// Init Connection
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:2151", os.Getenv("OCELOT_HOST")))
	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Fatal("Failed Connect to Server")
		return &WorkerPool{}, err
	}

	// Return Connection Object
	return &WorkerPool{
		Connection: &conn,
		Params:     wpParams,
	}, nil
}

// AcceptWork - Listens for work coming from server...
func (oc *WorkerPool) AcceptWork(ctx context.Context, cancel context.CancelFunc) {

	// Variables instantiated for each Client-> Server connection...
	var (
		termChan = make(chan os.Signal, 1)
		enc      = gob.NewEncoder(*oc.Connection)
		errChan  = make(chan error, 10)
		j        Job
		dec      = gob.NewDecoder(*oc.Connection) // BAD!
	)

	// The Decoder reads data from server and unmarsals into a Job object
	// Jobs are sent to workers as available...
	go func() {
		for {
			// [NOTE 02-08-2021] - This section was the problem area
			// that was dropping connections w. EOF; keep an eye out...
			errChan <- dec.Decode(&j)
		}
	}()

	// Logic for processing incoming requests...
	// Shutdown involves closing the client side jobs channel
	go func() {

		for {
			select {

			// Recieves from `errChan <- dec.Decode(&j)` above; will recieve nil if
			// Job properly marshalled...
			case err := <-errChan:
				if (err != nil) && (err != io.EOF) {
					// Either Server is sending bad Jobs
					//	- In which case the Encoder Buffer cannot be recovered -> Exit
					// 	- Or Client has been closed, buffer is incomplete -> Exit
					log.WithFields(log.Fields{"Error": err}).Error("Failed to Unmarshal Job From Server")
					cancel()
				}

				if err == io.EOF {
					// Server has shutdown (or client otherwise recieves no data from the Server?)
					// No more jobs to process...
					log.WithFields(
						log.Fields{
							"Error":              err,
							"Connection - Local": (*oc.Connection).LocalAddr().String(),
						},
					).Error("No Data Received - Server Shutdown")
					cancel()
				}

				// No Errors -> Send Job to WorkerPool and continue processing data...
				log.WithFields(
					log.Fields{"Job ID": j.ID, "Instance ID": j.InstanceID},
				).Info("Workers Recieved Job")
				oc.Params.JobChan <- j

			// [TODO, FINISH COMMENT] - Recieves a Cancelfunc() call; either from
			// the case(s) above, or ????
			case <-ctx.Done():
				log.Info("Deferred Cleanup -- TCP Listen")
				(*oc.Connection).Close()
				close(oc.Params.JobChan)
				return

			default:
			}

		}
	}()

	// Logic For sending results back to the server, listens on the worker's results channel.
	// While oc.Workers.StatusChan remains open (i.e. as long as workers are producing results)
	// encode those results to gob and send back to server
	go func() {
		for {
			select {

			case status := <-oc.Params.StatusChan:
				err := enc.Encode(&status)
				if err != nil {
					// If the job status is not encodeable; then exit with cancellation...
					log.WithFields(
						log.Fields{
							"Connection - Local":  (*oc.Connection).LocalAddr().String(),
							"Connection - Remote": (*oc.Connection).RemoteAddr().String(),
							"Error":               err,
						},
					).Error("TCP Error")
					cancel()
					return
				}

			// If cancellation, close the connection and exit
			case <-ctx.Done():
				log.Info("Post Context -- TCP Send")
				(*oc.Connection).Close()
				return

			default: // To make non-blocking...

			}
		}
	}()

	// Manual Cancellation
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(
		log.Fields{
			"Connection - Local":  (*oc.Connection).LocalAddr().String(),
			"Connection - Remote": (*oc.Connection).RemoteAddr().String(),
		},
	).Warn("Manual Shutdown")
	cancel()

	return
}

// Close - Wrapper around the net.Conn close function
func (oc *WorkerPool) Close() {
	log.WithFields(
		log.Fields{
			"Connection - Local":  (*oc.Connection).LocalAddr().String(),
			"Connection - Remote": (*oc.Connection).RemoteAddr().String(),
		},
	).Warn("Connection Closed")

	(*oc.Connection).Close()
}
