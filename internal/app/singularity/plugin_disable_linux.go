package singularity

import "github.com/sylabs/singularity/internal/pkg/plugin"

// DisablePlugin disables the named plugin
func DisablePlugin(name, libexecdir string) error {
	return plugin.Disable(name, libexecdir)
}
