// Package ocelot ...
package ocelot

import (
	"fmt"
	"math/rand"
	"ocelot/pkg/ocelot"
	"time"

	log "github.com/sirupsen/logrus"
)

// HTTPFail -
func HTTPFail(j *ocelot.JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	log.Info("Job Complete")
	return fmt.Errorf("Default Error")
}

// HTTPSuccess -
func HTTPSuccess(j *ocelot.JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	log.Info("Job Complete")
	return nil
}

// HTTPRandomSuccess -
func HTTPRandomSuccess(j *ocelot.JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	if 5 > rand.Intn(100) {
		return fmt.Errorf("Default Error")
	}
	log.Info("Job Complete")
	return nil

}
