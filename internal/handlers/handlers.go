// Package ocelot ...
package ocelot

import (
	ocelot "github.com/dmw2151/ocelot"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// S3Handler - Handler for Managing S3 Downloads
type S3Handler struct {
	Client *session.Session
}

// Work - Required to Implement Handler Interface
func (sh *S3Handler) Work(ji *ocelot.JobInstance) error {
	var err error

	// Init Service on reach Req, Recycle Underlying Session
	svc := s3.New(
		session.Must(sh.Client, err),
	)

	// In this case; the values in ji.Job.Params take precidence
	// over ji.Job.Path
	output, err := svc.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String(ji.Job.Params["Bucket"].(string)),
			Key:    aws.String(ji.Job.Params["Key"].(string)),
		},
	)

	if err != nil {
		log.Warnf("Failed to Download: %e", err)
		return err
	}

	// Other Logic Implemented here...
	log.Infof("Downloaded: %d Bytes", output.ContentLength)
	return nil
}

// MockHTTPHandler - For Testing HTTP Calls
type MockHTTPHandler struct{}

// Work - Required to Implement Handler Interface
// Not an HTTP Call; Just Waits 50-300ms for timing...
func (m *MockHTTPHandler) Work(ji *ocelot.JobInstance) error {
	time.Sleep(
		time.Duration(rand.Intn(250)+50.0) * time.Millisecond,
	)
	return nil
}
