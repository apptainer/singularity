package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/sylog"
)

// FuseInfo stores the FUSE-related information required or provided by
// plugins implementing options to add FUSE filesystems in the container.
type FuseInfo struct {
	DevFuseFd  int      // the file descriptor used to access the FUSE mount.
	MountPoint string   // the mount point for the FUSE filesystem.
	Program    []string // the FUSE driver program and all required arguments.
}

// GetPluginFuseMounts returns the list of plugins which have a valid
// FUSE configuration.
//
// In this context "valid" means that both the mount point and the FUSE
// driver program exist in the configuration. This function DOES NOT
// check if the /dev/fuse file descriptor has been assigned.
func (c *Common) GetPluginFuseMounts() []string {
	var list []string

	for name, raw := range c.Plugin {
		var info struct {
			Fuse FuseInfo
		}
		if err := json.Unmarshal(raw, &info); err != nil {
			// do not log anything because this would
			// introduce a lot of noise into the log, even
			// for the debug level
			continue
		}
		if len(info.Fuse.Program) > 0 && info.Fuse.MountPoint != "" {
			// This a valid configuration
			list = append(list, name)
		}
	}

	sort.Strings(list)
	return list
}

// SetFuseMount takes input from --fusemount options and creates plugin
// objects from them to hook in to the fuse plugin support code.
func (c *Common) SetFuseMount(fusemount []string) error {
	for _, mountspec := range fusemount {
		words := strings.Fields(mountspec)

		if !strings.HasPrefix(words[0], "container:") {
			return fmt.Errorf("fusemount spec does not begin with 'container:': %s", words[0])
		}
		words[0] = strings.Replace(words[0], "container:", "", 1)

		if len(words) == 1 {
			return fmt.Errorf("no whitespace separators found in command")
		}

		// The last word in the list is the mount point
		mnt := words[len(words)-1]
		words = words[0 : len(words)-1]

		var cfg struct {
			Fuse FuseInfo
		}

		cfg.Fuse.MountPoint = mnt
		cfg.Fuse.Program = words

		// Choose a name that makes sure they get used in alphabetical
		//  order so the mountpoints stay in order.  Assumes no more
		//  than 1000 plugins.
		pluginName := fmt.Sprintf("_fusemount%03d", len(c.Plugin))

		sylog.Verbosef("Mounting FUSE filesystem with %s %s as %s\n",
			strings.Join(words, " "), mnt, pluginName)

		if err := c.SetPluginConfig(pluginName, cfg); err != nil {
			return fmt.Errorf("could set plugin configuration: %v", err)
		}
	}
	return nil
}

// SetPluginFuseFd sets the /dev/fuse file descriptor fd for the
// specified plugin.
//
// This function tries to make sure that any additional configuration
// already found in the "Fuse" object is preserved.
func (c *Common) SetPluginFuseFd(name string, fd int) error {
	raw, found := c.Plugin[name]
	if !found {
		// named plugin does not have a configuration
		// entry, error out
		return errors.New("plugin not found")
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		// cannot unmarshal value as JSON, error out
		return errors.New("invalid JSON entry")
	}

	tmp, found := obj["Fuse"]
	if !found {
		// object does not have a Fuse key, error out
		return errors.New("missing Fuse JSON object")
	}

	info, ok := tmp.(map[string]interface{})
	if !ok {
		// invalid value, error out
		return errors.New("invalid Fuse JSON object")
	}

	info["DevFuseFd"] = fd
	obj["Fuse"] = info
	newval, err := json.Marshal(obj)
	if err != nil {
		// this should not happen
		return errors.New("cannot marshal new JSON object")
	}

	c.Plugin[name] = newval

	return nil
}
