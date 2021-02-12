// Package ocelot ...
package ocelot

import (
	"context"
	"encoding/gob"
	"net"

	//"os"
	"fmt"
	//"time"

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

// handleConnection - Handles incoming connections from workers
// Forwards jobInstances from the Producer's JobsChan to a worker. Encodes
// jobs using `gob` and sends jobInstance over TCP conn.
func (p *Producer) handleConnection(ctx context.Context, c net.Conn) {

	// Shutdown the followig resources on exit
	defer func() {
		log.Info("Worker Connection Being Terminated...")
		c.Close()
		p.Sem.Release(1)
		p.NOpenConnections--
	}()

	// TODO: Set Connection Type With First Message
	// Flush & Then Proceed to route Jobs Properly
	var handlerType string

	dec := gob.NewDecoder(c)
	_ = dec.Decode(&handlerType)

	log.WithFields(
		log.Fields{
			"Worker Addr": c.RemoteAddr().String(),
			"handlerType": handlerType,
		},
	).Info("New Connection")

	// Create encoder for each open connection;
	enc := gob.NewEncoder(c)

	for {
		select {
		// Work is available from the Producer's ticks
		case j := <-p.JobPool.JobChan[handlerType]:
			// This worker was shutdown after reciving the job, but the
			// connection was not yet closed. Send Jobs terminated by
			// connection drop back into the queue, this will almost
			// assuredly be an issue eventually
			err := enc.Encode(&j)

			if err != nil {
				log.WithFields(
					log.Fields{
						"Error":       err,
						"Job ID":      j.Job.ID,
						"Instance ID": j.InstanceID,
					},
				).Warnf("Failed to Dispatch Job - Worker Timeout (??) Retrying")

				// Only put back in queue if this won't cause an infinite loop...
				// I.e. There is another open connection to recieve this data...
				if p.NOpenConnections > 1 {
					p.JobPool.JobChan[handlerType] <- j
				}
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

		default:
			continue
		}
	}
}

// Serve --
func (p *Producer) Serve(ctx context.Context) error {

	var newConn = make(chan int, 1) // to manage communication

	// Start Jobs on Server Start, prevents filled queues on start...
	for _, j := range p.JobPool.Jobs {
		go j.startSchedule(ctx)
	}

	// Register Gather Operation for Intermediate Channels
	p.JobPool.gatherJobs()

	for {
		// Accept Incoming Connections; Single threaded through here...
		c, err := p.Listener.Accept()

		if c == nil {
			continue // Handle for No Connection Exists...
		}

		if (err != nil) && (c != nil) {
			log.WithFields(log.Fields{"Error": err}).Error("Rejected Connection")
			c.Close()
		}

		log.WithFields(
			log.Fields{"Worker Addr": c.RemoteAddr().String()},
		).Debug("Attempted Connection")

		// Otherwise, Assign connection && increment
		if !p.Sem.TryAcquire(1) {
			log.WithFields(
				log.Fields{
					"Permitted Conns": p.Config.MaxConnections,
					"Current Conns":   p.NOpenConnections,
				},
			).Debug("Too Many Connections")
			c.Close()
		} else {
			// NOTE: Continue to use NOpenConnections for Debug;
			// Should be same as Sempahore.size...
			p.OpenConnections[p.NOpenConnections] = &c
			p.NOpenConnections++
			newConn <- 1
		}

		// Chose one of the following cases && handle
		select {
		case <-newConn:
			go p.handleConnection(ctx, c)

		case <-ctx.Done(): // Exit if cancelled..
			return nil

		default:
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
	p.Listener.Close()

	// This should Only Close If not alreay closed??
	// Close all Channels...
	for _, v := range p.JobPool.JobChan {
		close(v)
	}

	p.NOpenConnections = 0

	return nil
}
