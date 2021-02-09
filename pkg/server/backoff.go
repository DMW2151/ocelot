// Package ocelot ...
package ocelot

import (
	"math"
	"time"
)

// BackoffFunc -
type BackoffFunc func(t time.Duration, i int) time.Duration

// BackoffPolicy - Not Yet Implemented
type BackoffPolicy struct {
	BackoffFunc BackoffFunc
	MaxRetries  int
	Retry       int
	C           chan bool
}

// ExpBackoff -
func ExpBackoff(t time.Duration, i int) time.Duration {
	// Function Logic for Exponential -> 2^n
	// In practice Base * 2^retry
	return t * time.Duration(math.Pow(2, float64(i)))
}
