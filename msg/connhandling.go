package ocelot

import (
	"context"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	// ErrOpenConnections - This error is reported...
	ErrOpenConnections = errors.New("no more connection pool slots")
)

// ConnPool - congrats- this is just a heavy channel...
// Used to manage incoming connections, allows N connections to
// `openConnections` based on the weight given to `sem`
//
// Record of open connections to Workers; Cannot Exceed Weight of
// Semaphore - For the Sake of the Core Lib, this can be (TYPE)
// but this is easily resolveable to []OcelotWorkerClient which
// is needed for msging
type ConnPool struct {
	Sem             *semaphore.Weighted
	OpenConnections []OcelotWorkerClient
}

// RegisterNewStreamingWorker - Adds a worker machine to the producer's connection
// pool that handles bi-directional stream based work
func RegisterNewStreamingWorker(addr string, cp ConnPool) OcelotWorker_ExecuteStreamClient {

	// Check if there are slots available, connect or return error and exit
	c, err := handleWorkerAssignment(addr, cp)
	if err != nil {
		// Cannot connect - Exit
		log.WithFields(
			log.Fields{"Err": err},
		).Warn("Worker Not Registered")
		return nil
	}

	stream, err := c.ExecuteStream(context.Background())
	if err != nil {
		// Cannot open stream - Exit
		log.WithFields(
			log.Fields{"Err": err},
		).Warnf("Failed to Open Stream")
		return nil
	}
	return stream
}

// SendtoWorker - blach
func SendtoWorker(sc OcelotWorker_ExecuteStreamClient, pChan <-chan *JobInstanceMsg) bool {

	var ji = &JobInstanceMsg{} // Dummy JobInstance

	// Start one goroutine to recieve content back from the worker, in this
	// case, the job status
	go func() {
		for {
			// While Data && !io.EOF, write to Producer side logs below
			// Could be (err == io.EOF) to be more permissive here; mostly io.EOF
			// or grpc.Terminaated errs...
			ji, err := sc.Recv()
			if err != nil {
				return
			}

			log.WithFields(
				log.Fields{
					"Success": ji.GetSuccess(),
					"CTime":   time.Unix(ji.GetCtime(), 0),
					"MTime":   time.Unix(ji.GetMtime(), 0),
					"Err":     err,
				},
			).Info("Recv Result")
		}
	}()

	// Start one goroutine to send jobs to the worker
	for {
		select {
		case ji = <-pChan:
			// If we recieve any error, INCLUDING EOF, Exit...
			if err := sc.Send(ji); err != nil {
				log.WithFields(
					log.Fields{"Err": err},
				).Warn("Failed to Send Message - Exiting")
				sc.CloseSend()
				return false
			}
		}
	}
}

func handleWorkerAssignment(addr string, cp ConnPool) (OcelotWorkerClient, error) {

	// Auth - Static TLS Config
	creds, err := credentials.NewClientTLSFromFile("certs/server.crt", "")
	if err != nil {
		log.Warn("Throw TLS Producer Side Init Error")
	}

	// Set Options Here
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStreamInterceptor(clientLoggingInterceptor),
	}

	// Producer Dials a Worker...
	conn, err := grpc.Dial(addr, opts...)
	// Returns some form of Net.Error/GRPC.Error re; dial failure
	// shouldn't be fatal, producer can have many workers...
	if err != nil {
		return nil, err
	}

	// Instatiates new NewOcelotWorkerClient; this is the meanignful
	// connection...
	c := NewOcelotWorkerClient(conn)
	if ok := cp.addObject(c); !ok {
		// No Connection Slots Remain - Specific Error
		return nil, errors.New("Whoopps - Sem lock")
	}

	return c, nil
}

func (cp *ConnPool) addObject(newc OcelotWorkerClient) bool {
	// TODO: Do Reflect Magic  Here...
	// Grab next empty slot available on the connPool
	if cp.Sem.TryAcquire(1) {
		for i, oc := range cp.OpenConnections {
			if oc == nil {
				cp.OpenConnections[i] = newc
				return true
			}
		}
	}
	log.Info("Could not Acquire Lock...")
	return false
}

// loggingInterceptor - Implement StreamServerInterceptor -
func clientLoggingInterceptor(
	ctx context.Context,
	desc *grpc.StreamDesc,
	cc *grpc.ClientConn,
	method string,
	streamer grpc.Streamer,
	opts ...grpc.CallOption,
) (grpc.ClientStream, error) {

	s, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		log.Fatalf("Interceptor Fail %+v", err)
	}

	// TODO: Implement Logging Interceptor Here...
	return s, nil
}
