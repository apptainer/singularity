// Copyright (c) 2020, Control Command Inc. All rights reserved.
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

const listLine = "%s\t%s\t%s\t%s\t%s\n"

// RemoteList prints information about remote configurations
func RemoteList(usrConfigFile string) (err error) {
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

	if err := syncSysConfig(c); err != nil {
		return err
	}

	// list in alphanumeric order
	names := make([]string, 0, len(c.Remotes))
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

	fmt.Println("Cloud Services Endpoints")
	fmt.Println("========================")
	fmt.Println()

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, listLine, "NAME", "URI", "ACTIVE", "GLOBAL", "EXCLUSIVE")
	for _, n := range names {
		sys := "NO"
		if c.Remotes[n].System {
			sys = "YES"
		}
		excl := "NO"
		if c.Remotes[n].Exclusive {
			excl = "YES"
		}
		active := "NO"
		if c.DefaultRemote != "" && c.DefaultRemote == n {
			active = "YES"
		}
		fmt.Fprintf(tw, listLine, n, c.Remotes[n].URI, active, sys, excl)
	}
	tw.Flush()

	if ep, err := c.GetDefault(); err == nil {
		if err := ep.UpdateKeyserversConfig(); err == nil {
			fmt.Println()
			fmt.Println("Keyservers")
			fmt.Println("==========")
			fmt.Println()

			tw = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", "URI", "GLOBAL", "INSECURE", "ORDER")
			order := 1
			for _, kc := range ep.Keyservers {
				if kc.Skip {
					continue
				}
				insecure := "NO"
				if kc.Insecure {
					insecure = "YES"
				}
				fmt.Fprintf(tw, "%s\tYES\t%s\t%d", kc.URI, insecure, order)
				if !kc.External {
					fmt.Fprintf(tw, "*\n")
				} else {
					fmt.Fprintf(tw, "\n")
				}
				order++
			}
			tw.Flush()

			fmt.Println()
			fmt.Println("* Active cloud services keyserver")
		}
	}

	if len(c.Credentials) > 0 {
		fmt.Println()
		fmt.Println("Authenticated Logins")
		fmt.Println("=================================")
		fmt.Println()

		tw = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(tw, "%s\t%s\n", "URI", "INSECURE")
		for _, r := range c.Credentials {
			insecure := "NO"
			if r.Insecure {
				insecure = "YES"
			}
			fmt.Fprintf(tw, "%s\t%s\n", r.URI, insecure)
		}
		tw.Flush()
	}

	return nil
}
