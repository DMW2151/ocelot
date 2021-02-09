// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"io"
	"net"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Server ...
type Server struct {
	Listener net.Listener
	Manager  *Manager
}

// Config -
type Config struct {
	Concurrency int
	Host        string
}

// NewServer - Create New Server, attempt to open connection
func NewServer(sConfig *Config, jConfigs []*JobConfig) *Server {

	// New Connection...
	l, err := net.Listen("tcp", sConfig.Host)
	if err != nil {
		log.WithFields(log.Fields{"Addr": sConfig.Host}).Fatal("Failed to Start Server", err)
	}

	log.WithFields(log.Fields{"Addr": sConfig.Host}).Info("Serving on")

	// New Manager
	jobs := make(map[uuid.UUID]*JobMeta)

	for _, conf := range jConfigs {
		j := newJobMeta(conf)
		jobs[j.ID] = j
	}

	m := &Manager{
		Jobs:    jobs,
		JobChan: make(chan *Job, sConfig.Concurrency),
	}

	// Combine to Create Server
	s := Server{
		Listener: l,
		Manager:  m,
	}

	return &s
}

// handleClientExit - Given a new connection, wait on the client to send
// an interupt message
//
// TODO: Check that connections can be closed without losing a job object...
func (s *Server) handleClientStatus(c net.Conn, clientExitChan chan<- int) {

	dec := gob.NewDecoder(c)

	var js = JobStatus{}

	// Blocking Call - Wait for Job Status...
	// Handle information coming in from Client...
	// Now you can do something with the jobStatus Object...
	// TODD - cHeck ordering here..
	for {
		err := dec.Decode(&js)

		// If not nil; but malformed
		// NOTE: Does this ever make sense...isn't the buffer screwed after this point??
		if (err != nil) && (err != io.EOF) {
			log.WithFields(log.Fields{"Error": err}).Warn("Received Invalid Job Status")
			clientExitChan <- 1
			return
		}

		// if data is EOF...exit
		if err == io.EOF {
			log.WithFields(
				log.Fields{
					"Connection - Local":  c.LocalAddr().String(),
					"Connection - Remote": c.RemoteAddr().String(),
				},
			).Warn("Server Recieved EOF; Null Data")
			clientExitChan <- 1
			return
		}

		// Job...
		j := s.Manager.Jobs[js.JobID]

		// Modify on Failure or True...
		if js.Success {
			// Update Exec Time
			log.WithFields(
				log.Fields{
					"Job ID":      js.JobID,
					"Instance ID": js.InstanceID,
				},
			).Info("Completed Job")

			// Reset Retry Status if Not at Base
			if j.BackoffPolicy.Retry != 0 {
				j.BackoffPolicy.Retry = 0
				j.Interval = j.BaseInterval
				j.Ticker = time.NewTicker(j.Interval) // Increase in GC; https://stackoverflow.com/questions/36689774/dynamically-change-ticker-interval
			}

		} else {
			// If Job Unmarshalled Properly, but with status failed, modify
			// backoff operator...

			// mutex lock this for the updated interval; consider the example
			// where the queue has 10 jobs for the same resource
			// all jobs in the queue try and fail at 00:00:00

			// They all update the backoff, from k * r^2 -> k * (r+1)^2 -> k * (r+2)^2
			// In practice, we'd want to see the next try at k * r^2
			log.WithFields(
				log.Fields{
					"Job ID":      js.JobID,
					"Instance ID": js.InstanceID,
				},
			).Info("Checking Lock")

			// Big Updaate Logic...
			go func() {
				if j.Sem.TryAcquire(1) {

					defer j.Sem.Release(1)

					// Got lock
					log.WithFields(
						log.Fields{
							"Job ID":      js.JobID,
							"Instance ID": js.InstanceID,
						},
					).Info("Got Lock")

					// Update Logic...
					if j.BackoffPolicy.Retry >= j.BackoffPolicy.MaxRetries {
						log.WithFields(
							log.Fields{
								"Job ID": js.JobID,
							},
						).Error("Job at Max Retries - Exiting")
						j.BackoffPolicy.C <- true
						return
					}

					j.BackoffPolicy.Retry++

					j.Interval = j.BackoffPolicy.BackoffFunc(j.BaseInterval, j.BackoffPolicy.Retry)
					j.Ticker = time.NewTicker(j.Interval) // Increase in GC; https://stackoverflow.com/questions/36689774/dynamically-change-ticker-interval

					log.WithFields(
						log.Fields{
							"Job ID":      js.JobID,
							"Instance ID": js.InstanceID,
							"Backoff":     j.Interval,
							"Locked For":  j.Interval - time.Millisecond*5,
						},
					).Info("Implementing Backoff")

					// set unlock...
					delay := time.NewTimer(j.Interval - time.Millisecond*5)
					<-delay.C
				}
			}()

		}
	}
}

// handleConnection - handle incoming connection...
func (s *Server) handleConnection(ctx context.Context, c net.Conn, clientExitChan chan int) {

	// Get a Job that has beend publishe from the JobsChan and write to
	// any TCP connection listening...
	enc := gob.NewEncoder(c)

	// On All Exit, must remove these...
	for {

		select {
		// Case Where Client Hasn't Sent Kill
		case j := <-s.Manager.JobChan:
			// Assigned to Worker...
			log.WithFields(
				log.Fields{
					"Worker": c.RemoteAddr().String(),
				},
			).Info("Dispatched Job")

			err := enc.Encode(&j)

			if err != nil {
				// Only Catches Nil Pointer Error; otherwise data -> client
				log.WithFields(
					log.Fields{
						"Error": err,
					},
				).Warn("Encoding Error")
				return
			}

		case <-clientExitChan:
			log.WithFields(
				log.Fields{"Addr": c.RemoteAddr().String()},
			).Info("Client Terminated")
			return

		// Server shutdown, might be uneeded...
		case <-ctx.Done():
			log.WithFields(
				log.Fields{"Addr": "127.0.0.1:2152"},
			).Info("Server Terminated - Got Kill Signal")
			return

		default:
		}
	}
}

// Serve --
// NOTES: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/
func (s *Server) Serve(ctx context.Context) error {

	newConnection := make(chan int, 1)
	clientExitChan := make(chan int, 1)

	for {
		// Accept Incoming Connections...
		conn, err := s.Listener.Accept()
		log.Info("Listening on: ", s.Listener.Addr().String(), &conn)

		if err != nil {
			log.Warn("Failed to Connect: ", err)

		} else {
			newConnection <- 1
		}

		// Split into Context Cancel Case and Dummy Sen...
		// Blocking...
		// No default...
		select {

		case <-newConnection:
			log.WithFields(
				log.Fields{"Addr": conn.RemoteAddr().String()},
			).Info("Accepted connection")

			// One to Listen -> Graceful Exit if client disconnects
			// One to Communicate Outwards...
			go s.handleConnection(ctx, conn, clientExitChan)
			go s.handleClientStatus(conn, clientExitChan)

		case <-ctx.Done():
			s.ShutDown(ctx)
		}
	}

}

// ShutDown -
// Shutdown works by first closing all open listeners, then closing all idle connections,
// and then waiting indefinitely for connections to return to idle and then shut down.
// If the provided context expires before the shutdown is complete, Shutdown returns
// the context’s error, otherwise it returns any error returned from closing the Server’s
// underlying Listener(s).
func (s *Server) ShutDown(ctx context.Context) {
	s.Listener.Close()
}
