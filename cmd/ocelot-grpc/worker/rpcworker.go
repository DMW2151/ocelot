package main

import (
	"os"
	"strconv"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	ocelot "github.com/dmw2151/ocelot"
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

	// Define S3 Handler
	h = ocelot.S3Handler{
		Session: s3Session,
		Client:  s3.New(session.Must(s3Session, err)),
	}

	wg sync.WaitGroup
)

func init() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05.0000"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	log.SetLevel(log.DebugLevel)
}

func main() {
	
	wp, _ := ocelot.NewWorkerPool(h)

	// In theory, you may want to block w. WG??
	wg.Add(1)
	wp.Serve(&wg)
	wg.Wait()

}
