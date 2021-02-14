package ocelot

import (
	"io/ioutil"
	"os"

	"github.com/dmw2151/ocelot"
	"gopkg.in/yaml.v2"
)

func OpenYaml(f string) *ocelot.ProducerConfig {

	var pCfg ocelot.ProducerConfig

	fi, _ := os.Open(f)
	data, _ := ioutil.ReadAll(fi)

	_ = yaml.Unmarshal([]byte(data), &pCfg)

	// Generate UUID if DNE...
	for _, j := range pCfg.Jobs {
		j.FromConfig(pCfg.JobChannelBuffer)
	}

	return &pCfg
}
