// Package ocelot ...
package ocelot

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// HTTPFail -
func HTTPFail(j *JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	log.Info("Job Complete")
	return fmt.Errorf("Default Error")
}

// HTTPSuccess -
func HTTPSuccess(j *JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	log.Info("Job Complete")
	return nil
}

// HTTPRandomSuccess -
func HTTPRandomSuccess(j *JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	if 5 > rand.Intn(100) {
		return fmt.Errorf("Default Error")
	}
	log.Info("Job Complete")
	return nil

}

// HTTPGetSize - Default Function to Test HTTP Calls
func HTTPGetSize(j *JobInstance) error {

	// Set Redirect to False
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make an HTTP Request from Job's path GET content
	// This is Where Some Logic Should Go
	resp, err := client.Get(j.Job.Path)

	if err != nil {
		log.WithFields(log.Fields{"Error": err}).Warn()
		return err
	}

	if resp.StatusCode != 200 {
		log.WithFields(
			log.Fields{"Status Code": resp.StatusCode},
		).Warn("Non-200 Status Code")
	}

	defer resp.Body.Close()

	return nil
}
