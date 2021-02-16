package ocelot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/sirupsen/logrus"
)

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
	if err := json.Unmarshal(cfgContent, v); err != nil {
		log.WithFields(
			log.Fields{"Config": fp},
		).Fatalf("Failed to Read Config Contents: %+v", err)
	}

	return v
}
