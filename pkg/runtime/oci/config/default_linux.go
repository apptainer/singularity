package config

func DefaultRuntimeOciConfig(cfg *RuntimeOciConfig) error {
	cfg.Version = &DefaultRuntimeOciVersion{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Hostname = &DefaultRuntimeOciHostname{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Root = &DefaultRuntimeOciRoot{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Annotations = &DefaultRuntimeOciAnnotations{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	cfg.Process = &DefaultRuntimeOciProcess{RuntimeOciSpec: &cfg.RuntimeOciSpec}
	return nil
}
