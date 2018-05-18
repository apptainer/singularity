/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package config

func DefaultRuntimeOciConfig(cfg *RuntimeOciConfig) error {
	cfg.Version = &DefaultRuntimeOciVersion{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Hostname = &DefaultRuntimeOciHostname{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Root = &DefaultRuntimeOciRoot{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Annotations = &DefaultRuntimeOciAnnotations{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Process = &DefaultRuntimeOciProcess{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	return nil
}
