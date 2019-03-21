// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/sylabs/singularity/internal/pkg/remote"
)

const listLine = "%s\t%s\t%s\n"

// RemoteList prints information about remote configurations
func RemoteList(usrConfigFile, sysConfigFile string) (err error) {
	c := &remote.Config{}

	// opening config file
	file, err := os.OpenFile(usrConfigFile, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no remote configurations")
		}
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	// read file contents to config struct
	c, err = remote.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing remote config data: %s", err)
	}

	if err := syncSysConfig(c, sysConfigFile); err != nil {
		return err
	}

	// list in alphanumeric order
	var names []string
	for n := range c.Remotes {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool {
		iName, jName := names[i], names[j]

		if c.Remotes[iName].System && !c.Remotes[jName].System {
			return true
		} else if !c.Remotes[iName].System && c.Remotes[jName].System {
			return false
		}

		return names[i] < names[j]
	})
	sort.Strings(names)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, listLine, "NAME", "URI", "SYS")
	for _, n := range names {
		uri := c.Remotes[n].URI
		if c.DefaultRemote != "" && c.DefaultRemote == n {
			n = fmt.Sprintf("[%s]", n)
		}

		sys := "NO"
		if c.Remotes[n].System {
			sys = "YES"
		}

		fmt.Fprintf(tw, listLine, n, uri, sys)
	}
	tw.Flush()
	return nil
}
