// Package ocelot ...
package ocelot

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// UnaryHandler - Interface that processes incoming values using unary
// method, I.E, One Request -> One Response
type UnaryHandler interface {
	Work(j *JobInstanceMsg, rCh chan<- *JobInstanceMsg) error
}

// StreamingHandler - Interface that processes incoming values using
// bi-directional streaming, N requests -> N responses w.o. promise
// of FIFO delivery
type StreamingHandler interface {
	Work(j *JobInstanceMsg, rCh chan<- *JobInstanceMsg) error
	StreamWork(s OcelotWorker_ExecuteStreamServer, rCh chan *JobInstanceMsg) error
}

// handleStreamData - For brevity in `StreamWork` methods, forwards
// stream data to the Work method
func handleStreamData(jh StreamingHandler, stream OcelotWorker_ExecuteStreamServer, rCh chan<- *JobInstanceMsg) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		_ = jh.Work(in, rCh)
	}
}

// NilHandler - Dummy Handler; Allows for Ping-Pong between GRPC client and
// server, does no work aside from marking mtime and success. Implements both
// UnaryHandler and StreamingHandler Interface
type NilHandler struct{}

// Work - Required to Implement JobHandler Interface(s)
func (nh NilHandler) Work(ji *JobInstanceMsg, rCh chan<- *JobInstanceMsg) error {
	return nil
}

// StreamWork - NilHandler's Streamwork handles the incoming streaming
// data and forwards the request to nh.Work
func (nh NilHandler) StreamWork(stream OcelotWorker_ExecuteStreamServer, rCh chan *JobInstanceMsg) error {
	if err := handleStreamData(nh, stream, rCh); err != nil {
		return err
	}
	return nil
}

// S3Handler - Handler for Managing S3 Downloads; Implements both UnaryHandler
// and StreamingHandler Interface
type S3Handler struct {
	Session *session.Session
	Client  *s3.S3
}

// Work - Required to Implement JobHandler Interface(s) - Downloas a File
func (sh S3Handler) Work(ji *JobInstanceMsg, rCh chan<- *JobInstanceMsg) error {

	// TODO: Init a new service on each request (if needed) while persisting the
	// underlying session in the struct (??)

	// In this case; the values in ji.Job.Params take precidence
	// over ji.Job.Path
	_, err := sh.Client.HeadObject(
		&s3.HeadObjectInput{
			// WARNING: Using Conversions from pb.any.Any -> Interface{} -> string
			// Can produce unexpected results, validate upstream that in  this
			// case params.bucket && params.key are string-like...
			Bucket: aws.String(string(ji.Params["bucket"])),
			Key:    aws.String(string(ji.Params["key"])),
		},
	)

	// Log failure to get headObject
	if err != nil {
		log.WithFields(
			log.Fields{
				"Bucket": string(ji.Params["bucket"]),
				"Key":    string(ji.Params["key"]),
				"Err":    err,
			},
		).Warn("Failed to Access S3 File")
		return err
	}

	/*
		TODO: Other Logic Implemented here - This can be Any Work
		with the file...
	*/

	// Mark Job Status Success
	ji.Success = true
	ji.Mtime = time.Now().UnixNano()

	rCh <- ji
	return nil
}

// StreamWork - S3Handler's StreamWork handles the incoming streaming
// data and forwards the request to sh.Work
func (sh S3Handler) StreamWork(stream OcelotWorker_ExecuteStreamServer, rCh chan *JobInstanceMsg) error {
	if err := handleStreamData(sh, stream, rCh); err != nil {
		return err
	}
	return nil
}
