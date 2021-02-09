// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"net"

	log "github.com/sirupsen/logrus"
)

// Producer ...
type Producer struct {
	Listener net.Listener
	JobPool  *JobPool
}

// ProducerConfig -
type ProducerConfig struct {
	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int

	// Addresses to Listen for New Connections
	ListenAddr string
}

// NewProducer - Create New Server, attempt to open a connection
func NewProducer(cfg *ProducerConfig, js []*Job) (*Producer, error) {

	// Listen on Addr...
	l, err := net.Listen("tcp", cfg.ListenAddr)

	if err != nil {
		log.WithFields(
			log.Fields{"ListenAddr": cfg.ListenAddr},
		).Fatal("Failed to Start Producer", err)
	}

	// Combine to create Producer object
	return &Producer{
		Listener: l,
		JobPool: &JobPool{
			Jobs:    js,
			JobChan: make(chan *JobInstance, cfg.JobChannelBuffer),
		},
	}, nil
}

// handleConnection - Handles incoming connections from workers
// Forwards jobInstances from the Producer's JobsChan to a worker. Encodes
// jobs using `gob` and sends jobInstance over TCP conn.
func (p *Producer) handleConnection(ctx context.Context, c net.Conn, clientExitChan chan int) {

	// Create encoder for each open connection; encoder never returns error (??)
	enc := gob.NewEncoder(c)

	// shutdown the followig resources on exit
	// TODO: - Set resources to clean up...

	for {
		// Select from one the the following...
		select {

		// Work is available from the Producer
		case j := <-p.JobPool.JobChan:

			log.WithFields(
				log.Fields{
					"Worker Address": c.RemoteAddr().String(),
					"Job ID":         j.Job.ID,
					"Instance ID":    j.InstanceID,
				},
			).Info("Dispatched Job")

			err := enc.Encode(&j)

			if err != nil {
				// Only catches nil pointer error; otherwise data sent
				// to worker as is. This can mean data loss on some objects
				// (e.g. You can't send a function over TCP, but a large struct
				// will serialize fine - even w. interface (??) check...)
				log.WithFields(log.Fields{"Error": err}).Warn("Encoding Error")
				return
			}

		// Signal that the Worker has terminated (either SIGTERM, Pipe Error
		// etc...)
		case <-clientExitChan:
			log.WithFields(
				log.Fields{"Worker Address": c.RemoteAddr().String()},
			).Warn("Connection Terminated")
			return

		// Producer shutdown, exit gracefully by closing outstandig
		// conections etc.
		case <-ctx.Done():
			log.WithFields(
				log.Fields{"Producer": c.LocalAddr().String()},
			).Info("Server Terminated - Got Kill Signal")
			return

		default: // No Blocking
		}
	}
}

// Serve --
// Reading: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/
func (p *Producer) Serve(ctx context.Context) error {

	// Create two dummy channels to manage communication
	newConn := make(chan int, 1)
	workerExit := make(chan int, 1)

	for {
		// Accept Incoming Connections...
		c, err := p.Listener.Accept()
		if err != nil {
			log.WithFields(log.Fields{"Error": err}).Info("Rejected Connection")
		} else {
			newConn <- 1
		}

		// Use a blocking select to chose one of the following cases
		// either new connection && handle as such, or context close
		select {

		case <-newConn:
			log.WithFields(
				log.Fields{"Worker Adddress": c.RemoteAddr().String()},
			).Info("New Connection")

			go p.handleConnection(ctx, c, workerExit)

		case <-ctx.Done():
			p.ShutDown(ctx)
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
