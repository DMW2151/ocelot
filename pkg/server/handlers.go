package ocelot

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// HTTPFail -
func HTTPFail(j Job) error {

	time.Sleep(100 * time.Millisecond)
	return fmt.Errorf("Default Error")
}

// HTTPSuccess -
func HTTPSuccess(j Job) error {

	time.Sleep(100 * time.Millisecond)
	return nil
}

// HTTPRandomSuccess -
func HTTPRandomSuccess(j Job) error {

	time.Sleep(100 * time.Millisecond)
	x0 := rand.Intn(100)
	if x0 > 14 {
		return fmt.Errorf("Default Error")
	}
	return nil

}

// HTTPGetSize - Default Function to Test HTTP Calls
func HTTPGetSize(j Job) error {

	// Set Redirect to False
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make an HTTP Request from Job's path GET content
	// This is Where Some Logic Should Go
	resp, err := client.Get(j.Path)

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

	// log.WithFields(
	// 	log.Fields{
	// 		"BodySize":    resp.ContentLength,
	// 		"Job ID":      j.ID,
	// 		"Instance ID": j.InstanceID,
	// 	},
	// ).Info("This Can be a Kinesis Put...")

	return nil
}
