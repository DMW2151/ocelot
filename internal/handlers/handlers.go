// Package ocelot ...
package ocelot

import (
	"fmt"
	"math/rand"
	"ocelot/pkg/ocelot"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	log "github.com/sirupsen/logrus"
)

// Initialize S3 Client

var s3Client, err = session.NewSession(
	&aws.Config{
		Region:                        aws.String("us-east-1"),
		CredentialsChainVerboseErrors: aws.Bool(true),
		Credentials:                   credentials.NewEnvCredentials(),
	},
)

// S3Grab -
// TOOD - Allow for more diversity in the handler func; e.g. an s3 put...
func S3Grab(ji *ocelot.JobInstance) error {

	svc := s3.New(session.Must(s3Client, err))

	output, err := svc.GetObject(
		&s3.GetObjectInput{
			Bucket: aws.String("ocelot-cctv"),
			Key:    aws.String("Screen Shot 2020-12-07 at 8.09.20 PM.png"),
		},
	)
	if err != nil {
		log.Warn(err)
		return err
	}

	log.Debug(*output.ContentLength)

	return nil
}

// HTTPRandomSuccess -
func HTTPRandomSuccess(ji *ocelot.JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	if 5 > rand.Intn(100) {
		return fmt.Errorf("Default Error")
	}
	log.Info("Job Complete")
	return nil

}

// HTTPFail -
func HTTPFail(ji *ocelot.JobInstance) error {
	time.Sleep(
		time.Duration(rand.Intn(450)+50.0) * time.Millisecond,
	)
	log.Info("Job Complete")
	return fmt.Errorf("Default Error")
}

// HTTPSuccess -
func HTTPSuccess(ji *ocelot.JobInstance) error {
	time.Sleep(
		time.Duration(rand.Intn(250)+50.0) * time.Millisecond,
	)
	return nil
}
