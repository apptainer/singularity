// +build singularity_engine

package loop

import (
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/util/singularityconf"
)

func GetMaxLoopDevices() int {
	// if the caller has set the current config use it
	// otherwise parse the default configuration file
	cfg := singularityconf.GetCurrentConfig()
	if cfg == nil {
		var err error

		configFile := buildcfg.SINGULARITY_CONF_FILE
		cfg, err = singularityconf.Parse(configFile)
		if err != nil {
			return 256
		}
	}
	return int(cfg.MaxLoopDevices)
}
