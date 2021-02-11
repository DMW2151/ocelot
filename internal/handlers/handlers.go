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

// HTTPRandomSuccess -
func HTTPRandomSuccess(ji *ocelot.JobInstance) error {
	time.Sleep(100 * time.Millisecond)
	if 5 > rand.Intn(100) {
		return fmt.Errorf("Default Error")
	}
	log.Info("Job Complete")
	return nil

}
