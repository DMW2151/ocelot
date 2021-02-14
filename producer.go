// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
)

// Producer - Object that Manages the Scheduler/Producer/Server
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

// TODO: Attach to Producer
var servKill = make(chan int, 1)

// decodeOneMesage - decode a single message from a connection
// Opens new decoder on a connection and returns one and only
// one message
func decodeOneMesage(c net.Conn) (msg string, err error) {
	dec := gob.NewDecoder(c)
	err = dec.Decode(&msg)
	if err != nil {
		return "", err
	}
	return msg, nil
}

// checkHandlerExists - does O(1) lookup to the JobChannels
// returns true if key exists
func (p *Producer) checkHandlerExists(msg string) bool {
	_, ok := p.jobPool.JobChan[msg]
	return ok
}

// validateConnection - Handles incoming connections before
// any work is assigned, rejects if server full
func (p *Producer) validateConnection(c net.Conn) bool {
	if p.sem.TryAcquire(1) {
		p.openConnections[p.nConn] = &c
		return true
	}

	log.WithFields(
		log.Fields{
			"Permitted Conns": p.config.MaxConn,
			"Current Conns":   p.nConn,
		},
	).Debug("Rejected Connection - Too Many Connections")
	c.Close()
	return false
}

// terminateConnection - Teardown for a connection; decriment
// count, release sem, and close connection
func (p *Producer) terminateConnection(c net.Conn) {
	p.sem.Release(1)
	p.nConn--
	c.Close()
	log.Warn("Worker Connection Being Terminated...")
}

// handleConnection - Handles incoming connections from workers
// Forwards jobInstances from JobChan to a worker.
func (p *Producer) handleConnection(ctx context.Context, c net.Conn) {

	defer p.terminateConnection(c)

	// First, recieve message from worker indicating what worker's
	// handler is; route jobs based on the recieved handler name
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

	// Create a new encoder to send job Instances over the wire
	// as reecieved
	enc := gob.NewEncoder(c)

	// TODO: This is unneeded overhead, find way to exclude this.
	// Create a minmal representation of the Instance struct to test
	// the decoder before sending a job. Do not want to dispatch a job
	// to a worker that's unable to recieve
	dec := gob.NewDecoder(c)

	// 1 Byte Struct "canary" is minimum struct that shares vals w. JobInstance
	var canary struct {
		Success bool
	}

	for {
		select {

		// On Cancel, exit this (and all other) open connections
		case <-ctx.Done():
			log.WithFields(
				log.Fields{"Producer Addr": c.LocalAddr().String()},
			).Error("Server Terminated - Got Kill Signal")
			return

		// On recieve tick
		case j := <-p.jobPool.JobChan[handlerType]:
			// Test read the value on the connection into canary, Set short
			// (BUT POSITIVE, NOT 0ms) read Timeout to force either i/o
			// timeout (do nothing) or closed pipe (shutdown an retry)
			c.SetReadDeadline(time.Now().Add(time.Millisecond * 5))
			if err := dec.Decode(&canary); err != nil {
				if netError, ok := err.(net.Error); ok && netError.Timeout() {
					// Do Nothing...
				} else { // TODO: Check if this can block
					log.WithFields(
						log.Fields{"Instance ID": j.InstanceID},
					).Warnf("Failed to Dispatch Job on Read (Retry): %v", err)
					p.jobPool.JobChan[handlerType] <- j
					return
				}
			}

			// Encode and send the jobInstance for worker to process
			err = enc.Encode(&j)
			log.WithFields(
				log.Fields{"Instance ID": j.InstanceID},
			).Info("Dispatched Job")

			if err != nil { // TODO: Check if this can block
				// If there is an error, recycle this job back into the channel
				log.WithFields(
					log.Fields{"Instance ID": j.InstanceID},
				).Warnf("Failed to Dispatch Job (Retry): %v ", err)
				p.jobPool.JobChan[handlerType] <- j
				return
			}
		}
	}
}

// Serve - Serve the producer, validate and accept all
// incoming worker connections
func (p *Producer) Serve(ctx context.Context, cancel context.CancelFunc) error {

	log.Info("Started Server")

	// Start all schedules on server start
	for _, j := range p.jobPool.Jobs {
		go p.jobPool.startSchedule(j)
	}

	// Call gather to forward each individual job's schedule to
	// intermediate channels for each handler type
	p.jobPool.gatherJobs()

	for {
		// Accept Incoming Connections; Single threaded through here...
		c, err := p.listener.Accept()
		if err != nil {
			select {
			case <-servKill:
				return nil
			default:
				// TODO: Implement Shutdown Logic Here...
				cancel()
			}
		}

		// Allow if open connection slot(s), otherwise boot
		if p.validateConnection(c) {
			go p.handleConnection(ctx, c)
		}
	}
}

// ShutDown - TODO: Roughly (more forcefully) emulate the behavior of the net/http
// close procedure, quoted below:
// 	- Close all open listeners
//	- Close all idle connections,
// 	- Wait indefinitely for connections to return to idle and then shut down.
func (p *Producer) ShutDown(ctx context.Context, cancel context.CancelFunc) error {

	// Call to cancel handles closing idle connections.
	// cancel() -> call to ctx.Done() for all connections which forces handleConnections
	// to exit, closing the connection & halting any more outgoing jobs
	cancel()

	// Send Break...
	close(servKill)

	// Close all open listeners
	p.listener.Close()

	// Stop all producers -
	// NOTE: current implementation of StopJob just pops the first job from the
	// List, so just call w. no args N times, where N is total num of jobs registered
	for range p.jobPool.Jobs {
		// Send QuitSig
		p.jobPool.StopJob()
	}
	time.Sleep(time.Millisecond * 1000)
	return nil
}
