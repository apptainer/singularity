package singularity

import "github.com/sylabs/singularity/internal/pkg/plugin"

// EnablePlugin enables the named plugin
func EnablePlugin(name, libexecdir string) error {
	return plugin.Enable(name, libexecdir)
}
