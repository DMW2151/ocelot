// Package ocelot -
package ocelot

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var exit chan bool

// WorkerPool - net.Conn with WorkParams attached. WorkParams
// set channel buffering, concurrency, etc. of WorkerPool
// TODO: Some reflect Magic to determine streaming or basic handler...
type WorkerPool struct {
	Listener  net.Listener
	Params    *WorkParams
	Pending   chan *JobInstanceMsg
	Results   chan *JobInstanceMsg
	RPCServer *grpc.Server
	// Most likely User defined function of type
	// JobHandler that workers in this pool execute
	Handler Handler
	mu      sync.Mutex
}

// Execute -
func (wp *WorkerPool) Execute(ctx context.Context, ji *JobInstanceMsg) (*JobInstanceMsg, error) {
	wp.Pending <- ji
	return ji, nil
}

// ExecuteStream -If you implement the interface, then cool, no need to define multiple methods for streaming
// for each hanler...
func (wp *WorkerPool) ExecuteStream(stream OcelotWorker_ExecuteStreamServer) error {
	if err := handleStreamData(wp.Handler, stream, wp.Results); err != nil {
		return err
	}
	return nil
}

// Serve - Listens for work coming from server...
func (wp *WorkerPool) Serve(ctx context.Context, cancel context.CancelFunc) {

	// Start Workers in the background and consume from pending channel
	// once a  producer is connected, they will push to pending...
	sessionUUID, _ := uuid.NewUUID()
	wp.startWorkers(ctx, sessionUUID)

	// Register as RPC Server - Needs an Execute Method; Whichh Should
	// Put a Value into the Channel...
	grpcServer := grpc.NewServer()
	RegisterOcelotWorkerServer(grpcServer, wp)

	log.Infof("Starting WorkerPool on: %s", wp.Listener.Addr())

	if err := grpcServer.Serve(wp.Listener); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}

}

// StartWorkers - Start Workers, runs `wp.Params.NWorkers`
func (wp *WorkerPool) startWorkers(ctx context.Context, sessionuuid uuid.UUID) {

	var wg sync.WaitGroup

	wg.Add(wp.Params.NWorkers)
	for i := 0; i < wp.Params.NWorkers; i++ {
		go wp.start(ctx, &wg)
	}

	log.WithFields(
		log.Fields{
			"Session ID": sessionuuid,
		},
	).Debugf("Started %d Workers", wp.Params.NWorkers)

	go func() { wg.Wait() }()
}

// start - Execute the Worker
// Write back to server; sends whenever job is done...
// Shouldn't encode multiple jobs before read..
func (wp *WorkerPool) start(ctx context.Context, wg *sync.WaitGroup) {

	log.Debug("Worker Started...")
	defer wg.Done()
	var k *JobInstanceMsg
	// TODO: Reflect Magic To Determine if Handler is Streaming
	// Or Basic
	for {
		k = <-wp.Pending
		wp.Handler.Work(k, wp.Results) // Do the Work; Call the Function...
	}

}
