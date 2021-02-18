package ocelot

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"crypto/sha256"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// Opens json file and passes into an interface - A pain to parse back to the
// concrete type (see `Workerparams.NewWorkerPool`)
func ParseConfig(fp string, v interface{}) interface{} {

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

// GenerateStaticUUID - Using a static UUID, generate a neww UUID from content fed
// to the function. Used for generating UUIDs for jobs that do not have UUID specified
// in config
func GenerateStaticUUID(b []byte) (uid uuid.UUID) {
	encUUID, _ := uuid.Parse("bcf3070f-7898-4399-bcae-4fcce2b451f5") // Static, but can be swapped for any new instance
	uid = uuid.NewHash(sha256.New(), encUUID, b, 3)
	return uid
}
