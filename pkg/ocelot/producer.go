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

	// Create encoder for each open connection;
	enc := gob.NewEncoder(c)

	for {
		select {
		// Work is available from the Producer's ticks
		case j := <-p.JobPool.JobChan:
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
					p.JobPool.JobChan <- j
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

		}
	}
}

// Serve --
// Reading: https://eli.thegreenplace.net/2020/graceful-shutdown-of-a-tcp-server-in-go/
// TODO: Closed Connection Cannot be Reopened Warning...
func (p *Producer) Serve(ctx context.Context) error {

	// Start Jobs on Server Start..
	// TODO (??): defer this until a connection is made available,
	// prevents throttle on start...
	for _, j := range p.JobPool.Jobs {
		go j.startSchedule(ctx)
	}

	// Register Gather Operation for Intermediate Channels
	p.JobPool.gatherJobs()

	// Create two dummy channels to manage communication
	newConn := make(chan int, 1)

	for {
		// Accept Incoming Connections; Single threaded through here...
		c, err := p.Listener.Accept()

		if c == nil {
			continue // Handle for No connection Exists...
		}

		if (err != nil) && (c != nil) {
			log.WithFields(log.Fields{"Error": err}).Error("Rejected Connection")
			c.Close()
		}

		log.WithFields(
			log.Fields{"Worker Addr": c.RemoteAddr().String()},
		).Debug("Attempted Connection")

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

		// Chose one of the following cases && handle as such
		// or context close
		select {
		case <-newConn:
			log.WithFields(
				log.Fields{"Worker Addr": c.RemoteAddr().String()},
			).Debug("New Connection")
			go p.handleConnection(ctx, c)

		case <-ctx.Done(): // If any location calls cancel
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
	// This should Only Close If not alreay closed...
	close(p.JobPool.JobChan)

	p.NOpenConnections = 0

	return nil
}
