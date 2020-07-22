// Copyright (c) 2019-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package starter

import (
	"fmt"

	"github.com/sylabs/singularity/internal/pkg/buildcfg"
	"github.com/sylabs/singularity/pkg/util/rlimit"
)

// copyConfigToEnv checks that the current stack size is big enough
// to pass runtime configuration through environment variables.
// On linux RLIMIT_STACK determines the amount of space used for the
// process's command-line arguments and environment variables.
func copyConfigToEnv(data []byte) ([]string, error) {
	var configEnv []string

	const (
		// space size for singularity argument and environment variables
		// this is voluntary bigger than the real usage
		singularityArgSize = 4096

		// for kilobyte conversion
		kbyte = 1024

		// DO NOT MODIFY those format strings
		envConfigFormat      = buildcfg.ENGINE_CONFIG_ENV + "%d=%s"
		envConfigCountFormat = buildcfg.ENGINE_CONFIG_CHUNK_ENV + "=%d"
	)

	// get the current stack limit in kilobytes
	cur, max, err := rlimit.Get("RLIMIT_STACK")
	if err != nil {
		return nil, fmt.Errorf("failed to determine stack size: %s", err)
	}

	// stack size divided by four to determine the arguments+environments
	// size limit
	argSizeLimit := (cur / 4)

	// config length to be passed via environment variables + some space
	// for singularity first argument
	configLength := uint64(len(data)) + singularityArgSize

	// be sure everything fit with the current argument size limit
	if configLength <= argSizeLimit {
		i := 1
		offset := uint64(0)
		length := uint64(len(data))
		for i <= buildcfg.MAX_ENGINE_CONFIG_CHUNK {
			end := offset + buildcfg.MAX_CHUNK_SIZE
			if end > length {
				end = length
			}
			configEnv = append(configEnv, fmt.Sprintf(envConfigFormat, i, string(data[offset:end])))
			if end == length {
				break
			}
			offset = end
			i++
		}
		if i > buildcfg.MAX_ENGINE_CONFIG_CHUNK {
			return nil, fmt.Errorf("engine configuration too big > %d", buildcfg.MAX_ENGINE_CONFIG_SIZE)
		}
		configEnv = append(configEnv, fmt.Sprintf(envConfigCountFormat, i))
		return configEnv, nil
	}

	roundLimitKB := 4 * ((configLength / kbyte) + 1)
	hardLimitKB := max / kbyte
	// the hard limit is reached, maybe user screw up himself by
	// setting the hard limit with ulimit or this is a limit set
	// by administrator, in this case returns some hints
	if roundLimitKB > hardLimitKB {
		hint := "check if you didn't set the stack size hard limit with ulimit or ask to your administrator"
		return nil, fmt.Errorf("argument size hard limit reached (%d kbytes), could not pass configuration: %s", hardLimitKB, hint)
	}

	hint := fmt.Sprintf("use 'ulimit -S -s %d' and run it again", roundLimitKB)

	return nil, fmt.Errorf(
		"argument size limit is too low (%d bytes) to pass configuration (%d bytes): %s",
		argSizeLimit, configLength, hint,
	)
}
