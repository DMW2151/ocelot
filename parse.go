package ocelot

import (
	"encoding/json"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

// Opens json file and passes into an interface - A pain to parse back to the
// concrete type (see `Workerparams.NewWorkerPool`)
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
	if err := json.Unmarshal(cfgContent, v); err != nil {
		log.WithFields(
			log.Fields{"Config": fp},
		).Fatalf("Failed to Read Config Contents: %+v", err)
	}

	return v
}
