// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"fmt"
	"net"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Producer - Object that Manages the Job-Side
type Producer struct {
	// Listener, to accept incoming connections
	listener net.Listener

	// JobPool, used to pool tickers from multiple jobs to one
	// see full description of struct's fields in jobpool.go
	jobPool *JobPool

	// Config, contains server level config information, see
	// full dedscription of struct's fields in producerparams.go
	config *ProducerConfig

	// Semaphore used to manage max connections available to
	// the object
	sem *semaphore.Weighted

	// Record and count of open connections...
	openConnections []*net.Conn
	nConn           int
}

// decodeOneMesage - decode a single message from a connection
func decodeOneMesage(c net.Conn) (msg string, err error) {
	dec := gob.NewDecoder(c)
	err = dec.Decode(&msg)
	if err != nil {
		return "", err
	}
	return msg, nil
}

// pokeEncoder - Poke Encoder to Ensure Is Valid.
func pokeDecoder(e *gob.Decoder, v interface{}) error {
	return e.Decode(v)
}

func (p *Producer) checkHandlerExists(msg string) bool {
	_, ok := p.jobPool.JobChan[msg]
	return ok
}

func (p *Producer) validateConnection(c net.Conn) bool {
	if p.sem.TryAcquire(1) {
		p.openConnections[p.nConn] = &c
		p.nConn++
		return true
	}

	log.WithFields(
		log.Fields{
			"Permitted Conns": p.config.MaxConn,
			"Current Conns":   p.nConn,
		},
	).Info("Rejected Connection - Too Many Connections")
	c.Close()
	return false
}

func (p *Producer) terminateConnection(c net.Conn) {
	log.Warn("Worker Connection Being Terminated...")
	p.sem.Release(1)
	p.nConn--
	c.Close()
}

// handleConnection - Handles incoming connections from workers
// Forwards jobInstances from the Producer's JobsChan to a worker. Encodes
// jobs using `gob` and sends jobInstance over TCP conn.
func (p *Producer) handleConnection(ctx context.Context, c net.Conn) {

	defer p.terminateConnection(c)

	// First, recieve message from worker indicating what its
	// attached handler is; then route jobs based on the recieved
	// handler name
	handlerType, err := decodeOneMesage(c)
	if err != nil {
		log.Warnf("Error on Decode Handler Type %v", err)
		return
	}

	// Do not use a connection slot on a worker servicing no jobs
	if !p.checkHandlerExists(handlerType) {
		log.Warnf("Handler Does Not Have Any Current Jobs - Booting")
		return
	}

	// Create an encoder to send job Instances over the wire
	// create a minmal representation of the Instance struct
	// to test the connection on each send...
	//var canary JobInstance

	var canary struct {
		InstanceID uuid.UUID
	}

	dec := gob.NewDecoder(c)
	enc := gob.NewEncoder(c)

	for {
		select {

		case <-ctx.Done():
			// On Cancel, exit this (and all other) open connections
			log.WithFields(
				log.Fields{"Producer Addr": c.LocalAddr().String()},
			).Error("Server Terminated - Got Kill Signal")
			return

		case j := <-p.jobPool.JobChan[handlerType]:
			// Canary Read Precedes Each Attempted Write...
			c.SetReadDeadline(time.Now().Add(time.Millisecond * 10))
			if err := pokeDecoder(dec, &canary); err != nil {
				if netError, ok := err.(net.Error); ok && netError.Timeout() {
					log.WithFields(
						log.Fields{
							"Instance ID": j.InstanceID,
							"Job":         fmt.Sprintf("%+v", canary),
						},
					).Tracef("Canary Read Timeout: %v", err)

				} else { // Some other error....
					log.WithFields(
						log.Fields{
							"Instance ID": j.InstanceID,
							"Job":         fmt.Sprintf("%+v", canary),
						},
					).Warnf("Canary Read Failed (Retry): %v", err)
					p.jobPool.JobChan[handlerType] <- j
					return
				}
			}

			// if canary.InstanceID.String() == "d5b0e6d3-523b-46cc-ad39-8a345acea4cb" {
			// 	log.Infof("Canary Did It's Job: %v", canary)
			// 	return
			// }

			// Encode and send jobInstance - Real....
			err = enc.Encode(&j)
			log.WithFields(
				log.Fields{"Instance ID": j.InstanceID},
			).Debugf("Dispatched Job: %v", err)
			// if err != nil {
			// 	// If there is an error, recycle this job back into the channel
			// 	log.WithFields(
			// 		log.Fields{"Instance ID": j.InstanceID},
			// 	).Warnf("Failed to Dispatch Job (Retrying): %v ", err)
			// 	p.jobPool.JobChan[handlerType] <- j
			// 	return
			// }

		}

	}

}

// Serve --
func (p *Producer) Serve(ctx context.Context) error {

	// Start all jobs on server start
	for _, j := range p.jobPool.Jobs {
		go p.jobPool.startSchedule(j)
	}

	// Register Gather Operation for Intermediate Channels
	p.jobPool.gatherJobs()

	for {
		// Accept Incoming Connections; Single threaded through here...
		c, err := p.listener.Accept()
		// c must be True, err must be False (reject if not c || err )
		if err != nil {
			log.Error("Rejected Connection: %v", err)
			c.Close()
		}

		// Chose one of the following cases && handle
		if p.validateConnection(c) {
			go p.handleConnection(ctx, c)
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
func (p *Producer) ShutDown(ctx context.Context, cancel context.CancelFunc) error {
	cancel()
	p.listener.Close()

	// This should Only Close If not alreay closed
	for _, v := range p.jobPool.JobChan {
		close(v)
	}

	p.nConn = 0

	return nil
}
