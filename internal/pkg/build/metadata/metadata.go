// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package metadata

import (
	"fmt"
	"strconv"
	"time"

	"github.com/sylabs/sif/pkg/sif"
	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/build/types"
)

// GetImageInfoLabels will make some image labels
func GetImageInfoLabels(labels map[string]map[string]string, fimg *sif.FileImage, b *types.Bundle) error {
	labels["system-partition"]["org.label-schema.schema-version"] = "1.0"

	// build date and time, lots of time formatting
	currentTime := time.Now()
	year, month, day := currentTime.Date()
	date := strconv.Itoa(day) + `_` + month.String() + `_` + strconv.Itoa(year)
	hour, min, sec := currentTime.Clock()
	time := strconv.Itoa(hour) + `:` + strconv.Itoa(min) + `:` + strconv.Itoa(sec)
	zone, _ := currentTime.Zone()
	timeString := currentTime.Weekday().String() + `_` + date + `_` + time + `_` + zone
	labels["system-partition"]["org.label-schema.build-date"] = timeString

	// singularity version
	labels["system-partition"]["org.label-schema.usage.singularity.version"] = buildcfg.PACKAGE_VERSION

	if fimg != nil {
		var err error
		// Get the primary partition data size
		primSize := make([]*sif.Descriptor, 1)
		primSize[0], _, err = fimg.GetPartPrimSys()
		if err != nil {
			return fmt.Errorf("failed getting main data: %s", err)
		}
		labels["system-partition"]["org.label-schema.image-size"] = readBytes(float64(primSize[0].Storelen))

		// Get the image arch
		imgParts, _, err := fimg.GetPartFromGroup(sif.DescrDefaultGroup)
		if err != nil {
			return fmt.Errorf("unable to get image part: %s", err)
		}

		if len(imgParts) != 1 {
			sylog.Warningf("Multiple partitions found, using first")
		}

		imageArch, err := imgParts[0].GetArch()
		if err != nil {
			return fmt.Errorf("unable to get image arch: %s", err)
		}
		labels["system-partition"]["org.label-schema.image-arch"] = sif.GetGoArch(cstrToString(imageArch[:]))
	}

	if b != nil {
		// help info if help exists in the definition and is run in the build
		if b.RunSection("help") && b.Recipe.ImageData.Help.Script != "" {
			labels["system-partition"]["org.label-schema.usage"] = "/.singularity.d/runscript.help"
			labels["system-partition"]["org.label-schema.usage.singularity.runscript.help"] = "/.singularity.d/runscript.help"
		}

		// bootstrap header info, only if this build actually bootstrapped
		if !b.Opts.Update || b.Opts.Force {
			for key, value := range b.Recipe.Header {
				labels["system-partition"]["org.label-schema.usage.singularity.deffile."+key] = value
			}
		}
	}

	return nil
}

// copy-paste from sylabs/sif
func cstrToString(str []byte) string {
	n := len(str)
	if m := n - 1; str[m] == 0 {
		n = m
	}
	return string(str[:n])
}

// TODO: put in a common package
func readBytes(in float64) string {
	i := 0
	size := in

	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB", "ZB", "YB"}

	for size > 1024 {
		size /= 1024
		i++
	}
	buf := fmt.Sprintf("%.*f %s", i, size, units[i])

	return buf
}
