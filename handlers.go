// Package ocelot ...
package ocelot

import (
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
)

// JobHandler - Interface for Object that processes
// a jobInstance
type JobHandler interface {
	Work(j *JobInstance) error
}

// S3Handler - Handler for Managing S3 Downloads
type S3Handler struct {
	Client *session.Session
}

// MockHTTPHandler - For Testing HTTP Calls
type MockHTTPHandler struct{}

// Work - Required to Implement Handler Interface
// WARNING: DO NOT KEEP THIS!!
func (sh S3Handler) Work(ji *JobInstance) error {
	return nil
	//var err error

	// Init Service on reach Req, Recycle Underlying Session
	// svc := s3.New(session.Must(sh.Client, err))

	// // In this case; the values in ji.Job.Params take precidence
	// // over ji.Job.Path
	// output, err := svc.GetObject(
	// 	&s3.GetObjectInput{
	// 		Bucket: aws.String(ji.Job.Params["bucket"].(string)),
	// 		Key:    aws.String(ji.Job.Params["key"].(string)),
	// 	},
	// )

	// if err != nil {
	// 	log.Warnf("Failed to Download: %e", err)
	// 	return err
	// }

	// // Other Logic Implemented here; why did this spit out...
	// log.Info(fmt.Sprint(*output.ContentLength))
	// return nil
}

// Work - Required to Implement Handler Interface
// Not an HTTP Call; Just Waits 50-300ms for timing...
func (m *MockHTTPHandler) Work(ji *JobInstance) error {
	time.Sleep(
		time.Duration(rand.Intn(250)+50.0) * time.Millisecond,
	)
	return nil
}
