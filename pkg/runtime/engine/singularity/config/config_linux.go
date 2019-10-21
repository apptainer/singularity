// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sylabs/singularity/internal/pkg/cgroups"
	"github.com/sylabs/singularity/internal/pkg/runtime/engine/config/oci"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/network"
	"github.com/sylabs/singularity/pkg/runtime/engine/config"
)

// EngineConfig stores both the JSONConfig and the FileConfig
type EngineConfig struct {
	JSON      *JSONConfig                `json:"jsonConfig"`
	OciConfig *oci.Config                `json:"ociConfig"`
	File      *config.FileConfig         `json:"-"`
	Network   *network.Setup             `json:"-"`
	Cgroups   *cgroups.Manager           `json:"-"`
	CryptDev  string                     `json:"-"`
	Plugin    map[string]json.RawMessage `json:"plugin"` // Plugin is the raw JSON representation of the plugin configurations
}

// FuseInfo stores the FUSE-related information required or provided by
// plugins implementing options to add FUSE filesystems in the
// container.
type FuseInfo struct {
	DevFuseFd  int      // the filedescritor used to access the FUSE mount
	MountPoint string   // the mount point for the FUSE filesystem
	Program    []string // the FUSE driver program and all required arguments
}

// NewConfig returns singularity.EngineConfig with a parsed FileConfig
func NewConfig() *EngineConfig {
	ret := &EngineConfig{
		JSON:      new(JSONConfig),
		OciConfig: new(oci.Config),
		File:      new(config.FileConfig),
		Plugin:    make(map[string]json.RawMessage),
	}

	return ret
}

// GetPluginConfig retrieves the configuration for the named plugin
func (e *EngineConfig) GetPluginConfig(plugin string, cfg interface{}) error {
	if tmp, found := e.Plugin[plugin]; found {
		return json.Unmarshal(tmp, cfg)
	}

	return nil
}

// SetPluginConfig sets the configuration for the named plugin
func (e *EngineConfig) SetPluginConfig(plugin string, cfg interface{}) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	e.Plugin[plugin] = json.RawMessage(data)
	return nil
}

//SetFuseMount takes input from --fusemount options and creates plugin objects
//  from them to hook in to the fuse plugin support code
func (e *EngineConfig) SetFuseMount(fusemount []string) error {
	if !e.File.EnableFusemount {
		sylog.Fatalf("--fusemount disabled by configuration")
	}

	for _, mountspec := range fusemount {
		words := strings.Fields(mountspec)

		if !strings.HasPrefix(words[0], "container:") {
			sylog.Fatalf("fusemount spec does not begin with 'container:': %s.\n", words[0])
		}
		words[0] = strings.Replace(words[0], "container:", "", 1)

		if len(words) == 1 {
			sylog.Fatalf("No whitespace separators found in command")
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
		pluginName := fmt.Sprintf("_fusemount%03d", len(e.Plugin))

		sylog.Verbosef("Mounting FUSE filesystem with %s %s as %s\n",
			strings.Join(words, " "), mnt, pluginName)

		if err := e.SetPluginConfig(pluginName, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Cannot set plugin configuration: %+v\n", err)
		}
	}
	return nil
}

// GetPluginFuseMounts returns the list of plugins which have a valid
// FUSE configuration.
//
// In this context "valid" means that both the mount point and the FUSE
// driver program exist in the configuration. This function DOES NOT
// check if the /dev/fuse file descriptor has been assigned.
func (e *EngineConfig) GetPluginFuseMounts() []string {
	var list []string

	for name, raw := range e.Plugin {
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

// SetPluginFuseFd sets the /dev/fuse file descriptor fd for the
// specified plugin
//
// This function tries to make sure that any additional configuration
// already found in the "Fuse" object is preserved.
func (e *EngineConfig) SetPluginFuseFd(name string, fd int) error {
	raw, found := e.Plugin[name]
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

	e.Plugin[name] = json.RawMessage(newval)

	return nil
}
