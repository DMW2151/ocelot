package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	msg "github.com/dmw2151/ocelot/msg"
	log "github.com/sirupsen/logrus"
)

var (
	// Define S3 Sessions
	s3Session, _ = session.NewSession(
		&aws.Config{
			Region:                        aws.String("us-east-1"),
			CredentialsChainVerboseErrors: aws.Bool(true),
			Credentials:                   credentials.NewEnvCredentials(),
		},
	)

	// Placeholder Error
	err error

	// Define S3 Handler - Included as part of ocelot/msg but could be anything
	// that satisfies ocelot/msg Handler interface...
	h = msg.S3Handler{
		Session: s3Session,
		Client:  s3.New(session.Must(s3Session, err)),
	}
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func main() {

	// Start New WorkerPool. Blocks forerver, waiting to execute
	// the handler, in this case (msg.S3Handler)
	wp, _ := msg.NewWorkerPool(h)
	wp.Serve()
}
