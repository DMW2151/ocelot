package ocelot

import (
	"io/ioutil"
	"net"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/semaphore"
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

	// List of Jobs to Init
	Jobs []*Job `yaml:"jobs"`
}

// Opens default server config at $OCELOT_CFG and creates a new producer config
// object, opens a new server w.
func openOcelotConfig() (pCfg ProducerConfig) {

	// Open config...
	cfgContent, err := ioutil.ReadFile(os.Getenv("OCELOT_CFG"))

	if err != nil {
		log.WithFields(
			log.Fields{"Config": os.Getenv("OCELOT_CFG")},
		).Fatalf("Failed to Read: %+v", err)
	}

	// Expand environment variables && read into config
	cfgContent = []byte(os.ExpandEnv(string(cfgContent)))
	if err := yaml.Unmarshal(cfgContent, &pCfg); err != nil {
		log.WithFields(
			log.Fields{"Config": os.Getenv("OCELOT_CFG")},
		).Fatalf("Failed to Read: %+v", err)
	}

	// Generate UUID Job if DNE:
	// WARNING: If Not Set In Config, Job UUIDs will vary between sessions
	for _, j := range pCfg.Jobs {
		j.FromConfig(pCfg.JobChannelBuffer)
	}

	return pCfg
}

// newListener from config...
func newListener(cfg *ProducerConfig) net.Listener {

	var l, err = net.Listen("tcp", cfg.ListenAddr)

	// Not Testable...Can't Ensure that Exits with Fail...
	if err != nil {
		log.WithFields(
			log.Fields{"Producer Addr": cfg.ListenAddr},
		).Fatal("Failed to Start Producer", err)
	}
	return l
}

// NewProducer - Create New Server, Initializes listener and
// Job Channels
func NewProducer(fp string) *Producer {

	cfg := openOcelotConfig()

	// Create Producer, filling in values required from config
	p := Producer{
		listener: newListener(&cfg),
		jobPool: &JobPool{
			Jobs:    cfg.Jobs,
			JobChan: make(map[string]chan (*JobInstance), cfg.JobChannelBuffer),
			wg:      sync.WaitGroup{},
		},
		openConnections: make([]*net.Conn, cfg.MaxConn),
		config:          &cfg,
		sem:             semaphore.NewWeighted(int64(cfg.MaxConn)),
	}

	// Init Channels for each Unique Handler...
	for _, j := range p.jobPool.Jobs {
		handlerType := j.Params["type"].(string)

		if _, ok := p.jobPool.JobChan[handlerType]; !ok {
			p.jobPool.JobChan[handlerType] = make(chan *JobInstance, cfg.JobChannelBuffer)
		}

	}

	return &p
}
