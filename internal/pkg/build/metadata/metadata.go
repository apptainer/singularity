// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package metadata

import (
	"strconv"
	"time"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/build/types"
)

// GetImageInfoLabels will make some image labels
func GetImageInfoLabels(labels map[string]string, b *types.Bundle) {
	labels["org.label-schema.schema-version"] = "2.0"

	// build date and time, lots of time formatting
	currentTime := time.Now()
	year, month, day := currentTime.Date()
	date := strconv.Itoa(day) + `_` + month.String() + `_` + strconv.Itoa(year)
	hour, min, sec := currentTime.Clock()
	time := strconv.Itoa(hour) + `:` + strconv.Itoa(min) + `:` + strconv.Itoa(sec)
	zone, _ := currentTime.Zone()
	timeString := currentTime.Weekday().String() + `_` + date + `_` + time + `_` + zone
	labels["org.label-schema.build-date"] = timeString

	// singularity version
	labels["org.label-schema.usage.singularity.version"] = buildcfg.PACKAGE_VERSION

	if b != nil {
		// help info if help exists in the definition and is run in the build
		if b.RunSection("help") && b.Recipe.ImageData.Help.Script != "" {
			labels["org.label-schema.usage"] = "/.singularity.d/runscript.help"
			labels["org.label-schema.usage.singularity.runscript.help"] = "/.singularity.d/runscript.help"
		}

		// bootstrap header info, only if this build actually bootstrapped
		if !b.Opts.Update || b.Opts.Force {
			for key, value := range b.Recipe.Header {
				labels["org.label-schema.usage.singularity.deffile."+key] = value
			}
		}
	}
}
