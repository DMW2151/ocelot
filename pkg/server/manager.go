// Package ocelot ...
// Notes
// 	- https://callistaenterprise.se/blogg/teknik/2019/10/05/go-worker-cancellation/
package ocelot

import (
	"github.com/google/uuid"
)

// Manager - Manages a queue of Jobs and Sends them out
// to Workers
type Manager struct {
	Jobs    map[uuid.UUID]*JobMeta
	JobChan chan *Job
}
