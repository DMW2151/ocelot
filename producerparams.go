package ocelot

import (
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// ProducerConfig - Factory Struct for Producer
type ProducerConfig struct {
	// Non-negative Integer representing the max jobs held in
	// Producer-side job channel
	JobChannelBuffer int `yaml:"job_channel_buffer"`

	// Addresses to Listen for New Connections - Supports
	// Standard IPV4 addressing conventions e.g.
	// [::]:2151, 127.0.0.1:2151, etc...
	ListenAddr string `yaml:"listen_addr"`

	// Max Active Connections to producer
	MaxConn int `yaml:"max_connections"`

	// Default Timeout for Sends to Staging Channnel
	JobTimeout time.Duration `yaml:"job_timeout"`

	// List of Jobs to Init on Server Start
	JobCfgs []*JobConfig `yaml:"jobs"`

	// Buffer for the pool of multiple jobs sharing one handler...
	HandlerBuffer int `yaml:"handler_buffer"`
}

// Opens default server config at $OCELOT_CFG and creates a new producer config
// object, opens a new server w.
// YUCK: pass *T as interface, unmarshal into v and then convert back to concrete type???
// 		cfg := parseConfig(
// 			os.Getenv("OCELOT_WORKER_CFG"),
// 			&WorkParams{},
// 		).(*WorkParams)
func parseConfig(fp string, v interface{}) interface{} {

	// Open config file...
	cfgContent, err := ioutil.ReadFile(fp)

	if err != nil {
		log.WithFields(
			log.Fields{"Config": fp},
		).Fatalf("Failed to Read Config: %+v", err)
	}

	// Expand environment variables && read into config
	cfgContent = []byte(os.ExpandEnv(string(cfgContent)))
	if err := yaml.Unmarshal(cfgContent, v); err != nil {
		log.WithFields(
			log.Fields{"Config": fp},
		).Fatalf("Failed to Read Config Contents: %+v", err)
	}

	return v
}
