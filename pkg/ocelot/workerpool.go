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

// WorkerPool - net connection with WorkParams attached..
// WorkParams set channel buffering, concurrency, etc. of
// WorkerPool
type WorkerPool struct {
	Connection *net.Conn
	Params     *WorkParams
	Pending    chan JobInstance
}

// NewWorkerPool - Factory function for creating a WorkerPool
func NewWorkerPool(wp *WorkParams) (*WorkerPool, error) {

	// Initialize Connection - See env var @ os.Getenv("OCELOT_HOST")
	c, err := net.Dial(
		"tcp",
		fmt.Sprintf("%s:%s", wp.Host, wp.Port),
	)
	if err != nil {
		log.WithFields(
			log.Fields{
				"Error": err,
				"Host":  wp.Host,
			}).Fatal("Failed Connect to Producer")
	}

	// Return WorkerPool Object from Config
	return &WorkerPool{
		Connection: &c,
		Params:     wp,
		Pending:    make(chan JobInstance, wp.MaxBuffer),
	}, nil
}

// AcceptWork - Listens for work coming from server...
func (wp *WorkerPool) AcceptWork(ctx context.Context, cancel context.CancelFunc) {

	// Variables instantiated for each Client-> Server connection...
	var (
		termChan = make(chan os.Signal, 1)
		errChan  = make(chan error, 10)
		j        JobInstance
		dec      = gob.NewDecoder(*wp.Connection) // BAD!
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
							"Connection - Local": (*wp.Connection).LocalAddr().String(),
						},
					).Error("No Data Received - Server Shutdown")
					cancel()
				}

				// No Errors -> Send Job to WorkerPool and continue processing data...
				log.WithFields(
					log.Fields{"Job ID": j.Job.ID, "Instance ID": j.InstanceID},
				).Info("Workers Recieved Job")
				wp.Pending <- j

			// [TODO, FINISH COMMENT] - Recieves a Cancelfunc() call; either from
			// the case(s) above, or ????
			case <-ctx.Done():
				log.Info("Deferred Cleanup -- TCP Listen")
				(*wp.Connection).Close()
				close(wp.Pending)
				return

			default:
			}

		}
	}()

	// Manual Cancellation
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(
		log.Fields{
			"Connection - Local":  (*wp.Connection).LocalAddr().String(),
			"Connection - Remote": (*wp.Connection).RemoteAddr().String(),
		},
	).Warn("Manual Shutdown")
	cancel()

	return
}

// Close - Wrapper around the net.Conn close function
func (wp *WorkerPool) Close() {
	log.WithFields(
		log.Fields{
			"Connection - Local":  (*wp.Connection).LocalAddr().String(),
			"Connection - Remote": (*wp.Connection).RemoteAddr().String(),
		},
	).Warn("Connection Closed")

	(*wp.Connection).Close()
}
