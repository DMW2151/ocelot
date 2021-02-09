// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"net"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Producer ...
type Producer struct {
	Listener         net.Listener
	JobPool          *JobPool
	Config           *ProducerConfig
	Sem              *semaphore.Weighted
	OpenConnections  []*net.Conn
	NOpenConnections int
}

// ProducerConfig -
type ProducerConfig struct {
	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int

	// Addresses to Listen for New Connections
	ListenAddr string

	// Max Active Connections to producer
	MaxConnections int
}

// NewProducer - Create New Server, attempt to open a connection
func NewProducer(cfg *ProducerConfig, js []*Job) (*Producer, error) {

	// Listen on Addr...
	l, err := net.Listen("tcp", cfg.ListenAddr)

	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": cfg.ListenAddr},
		).Fatal("Failed to Start Producer", err)
	}

	log.WithFields(
		log.Fields{"Producer Addr": cfg.ListenAddr},
	).Info("Listening")

	// Combine to create Producer object
	return &Producer{
		Listener:         l,
		JobPool:          &JobPool{Jobs: js, JobChan: make(chan *JobInstance, cfg.JobChannelBuffer)},
		OpenConnections:  make([]*net.Conn, cfg.MaxConnections),
		NOpenConnections: 0,
		Config:           cfg,
		Sem:              semaphore.NewWeighted(int64(cfg.MaxConnections)),
	}, nil
}

// handleConnection - Handles incoming connections from workers
// Forwards jobInstances from the Producer's JobsChan to a worker. Encodes
// jobs using `gob` and sends jobInstance over TCP conn.
func (p *Producer) handleConnection(ctx context.Context, c net.Conn, clientExitChan chan int) {

	// Shutdown the followig resources on exit
	defer func() {
		c.Close()
		p.Sem.Release(1)
		p.NOpenConnections--
	}()

	// Create encoder for each open connection; encoder never returns error (??)
	enc := gob.NewEncoder(c)

	for {
		// Select from one the the following...
		select {
		// Work is available from the Producer
		case j := <-p.JobPool.JobChan:
			err := enc.Encode(&j) // Chances that connection drops riiight here are small...

			if err != nil {
				// Catches nil pointer error; otherwise data sent to worker as is.
				// Will also catch broken pipe error etc.
				log.WithFields(
					log.Fields{
						"Error":       err,
						"Job ID":      j.Job.ID,
						"Instance ID": j.InstanceID,
					},
				).Warnf("Failed to Dispatch Job, Retrying")

				// NOTE: RETRY (??) - Send Jobs terminated by connection drop
				// back into the queue, this will almost assuredly be an issue eventually
				p.JobPool.JobChan <- j

				return
			}

			log.WithFields(
				log.Fields{
					"Worker Addr": c.RemoteAddr().String(),
					"Job ID":      j.Job.ID,
					"Instance ID": j.InstanceID,
				},
			).Debug("Dispatched Job")

		// Producer shutdown, exit gracefully by closing outstandig
		// conections etc.
		case <-ctx.Done():
			log.WithFields(
				log.Fields{"Producer Addr": c.LocalAddr().String()},
			).Error("Server Terminated - Got Kill Signal")
			return

		default: // No Blocking
		}
	}
}

// Serve --
// Reading: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/
func (p *Producer) Serve(ctx context.Context) error {

	// Close...
	defer func() {
		p.Listener.Close()
	}()

	// Start Jobs on Server Start..
	// TODO (??): defer this until a connection is made available,
	// prevents throttle on start...
	for _, j := range p.JobPool.Jobs {
		go j.StartSchedule(ctx, p.JobPool.JobChan)
	}

	// Create two dummy channels to manage communication
	newConn := make(chan int, 1)
	workerExit := make(chan int, 1)

	for {
		// Accept Incoming Connections; Single threaded through here...
		c, err := p.Listener.Accept()
		log.WithFields(log.Fields{"Worker Addr": c.RemoteAddr().String()}).Debug("Attempted Connection")

		if err != nil {
			log.WithFields(log.Fields{"Error": err}).Error("Rejected Connection")
			c.Close()
		}

		// Otherwise, Assign connection && increment - (Need Mutex??)
		if !p.Sem.TryAcquire(1) {
			// Log Connection Pool Status...
			log.WithFields(
				log.Fields{
					"Permitted Conns": p.Config.MaxConnections,
					"Current Conns":   p.NOpenConnections,
				},
			).Debug("Too Many Connections")
			c.Close()
		} else {
			// NOTE: Continue to use NOpenConnections for Debug; Should Always
			// be in line w. Sempahore.size...
			p.OpenConnections[p.NOpenConnections] = &c
			p.NOpenConnections++
			newConn <- 1
		}

		// Use a blocking select to chose one of the following cases
		// either new connection && handle as such, or context close
		select {
		case <-newConn:
			log.WithFields(
				log.Fields{"Worker Addr": c.RemoteAddr().String()},
			).Debug("New Connection")
			go p.handleConnection(ctx, c, workerExit)

		case <-ctx.Done():
			p.ShutDown(ctx)
		default: // WARNING: Added 02-09-2021, Careful!!; Appears to have Worked
		}
	}
}

// ShutDown - Very roughly emulates the behavior of the net/http close procedure;
// quoted below:
// 		Shutdown works by first closing all open listeners, then closing all idle connections,
// 		and then waiting indefinitely for connections to return to idle and then shut down.
// 		If the provided context expires before the shutdown is complete, Shutdown returns
// 		the context’s error, otherwise it returns any error returned from closing the Server’s
// 		underlying Listener(s).
// TODO: See above...
func (p *Producer) ShutDown(ctx context.Context) {
	p.Listener.Close()
	close(p.JobPool.JobChan)
}
