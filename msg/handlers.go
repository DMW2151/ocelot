package ocelot

import (
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// Handler - Interface that processes incoming values using unary
// method, I.E, One Request -> One Response
type Handler interface {
	Work(j *JobInstanceMsg, rCh chan *JobInstanceMsg) error
}

var (
	// ErrIncompleteRequest - This error is reported when....
	ErrIncompleteRequest = errors.New("some handler specific request params not met")
)

// NilHandler - Dummy Handler; Allows for Ping-Pong between GRPC client and
// server, does no work aside from marking mtime and success. Implements both
// UnaryHandler and StreamingHandler Interface
type NilHandler struct{}

// S3Handler - Handler for Managing S3 Downloads; Implements both UnaryHandler
// and StreamingHandler Interface
type S3Handler struct {
	Session *session.Session
	Client  *s3.S3
}

// Work - Required to Implement JobHandler Interface(s)
func (nh NilHandler) Work(ji *JobInstanceMsg, rCh chan *JobInstanceMsg) error {
	return nil
}

// Work - Required to Implement JobHandler Interface(s) - Downloas a File
func (sh S3Handler) Work(ji *JobInstanceMsg, rCh chan *JobInstanceMsg) error {

	params := ji.GetParams()
	if params == nil {
		// TODO - Malformed Header or Params Not Supplied
		return ErrIncompleteRequest
	}

	_, err := sh.Client.HeadObject(
		&s3.HeadObjectInput{
			// WARNING: Using Conversions from pb.any.Any -> Interface{} -> string
			// Can produce unexpected results, validate upstream that in  this
			// case params.bucket && params.key are string-like...
			Bucket: aws.String(string(params["bucket"])),
			Key:    aws.String(string(params["key"])),
		},
	)

	// NOTE: Check on success status, msg.success is false on init,
	// no need to re-mark as false...
	if err != nil {
		// TODO: Send Specific Errors back to the Producer...
		log.WithFields(
			log.Fields{
				"Bucket": string(params["bucket"]),
				"Key":    string(params["key"]),
				"Err":    err,
			},
		).Warn("Failed to Access S3 File")
	} else {
		ji.Success = true
	}

	rCh <- ji
	ji.Mtime = time.Now().UnixNano()
	return err
}

// handleStreamData - For brevity in `StreamWork` methods, forwards
// stream data to the Work method
func handleStreamData(jh Handler, stream OcelotWorker_ExecuteStreamServer, rCh chan *JobInstanceMsg) error {
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				//return io.EOF // or nil (?)
			}
			if err != nil {
				log.Errorf("Stream Terminate by Producer: %+v", err)
				return
			}

			_ = jh.Work(in, rCh)
		}
	}()

	// Send Response Back to Manager - Recieve from rCh, where
	// Work is sent once completed
	for {
		if err := stream.Send(<-rCh); err != nil {
			log.Errorf("Return Result to Producer Failed: %+v", err)
			return err
		}
	}
}

// loggingInterceptor - Implement StreamServerInterceptor -
// Server side logging for confirming incoming requests
func loggingInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {

	// TODO: Implement Logging Interceptor Here...
	log.Debug("Stream Recieved...")

	// Forward Content To StreamExecutor
	if err := handler(srv, ss); err != nil {
		return err
	}

	return nil
}
