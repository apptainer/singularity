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
	"strings"
	"text/tabwriter"

	"github.com/sylabs/singularity/internal/pkg/remote"
	"github.com/sylabs/singularity/internal/pkg/remote/endpoint"
	"github.com/sylabs/singularity/pkg/sylog"
)

const statusLine = "%s\t%s\t%s\t%s\n"

type status struct {
	name    string
	uri     string
	status  string
	version string
}

// RemoteStatus checks status of services related to an endpoint
// If the supplied remote name is an empty string, it will attempt
// to use the default remote.
func RemoteStatus(usrConfigFile, name string) (err error) {
	c := &remote.Config{}

	if name != "" {
		sylog.Infof("Checking status of remote: %s", name)
	} else {
		sylog.Infof("Checking status of default remote.")
	}

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

	var e *endpoint.Config
	if name == "" {
		e, err = c.GetDefault()
	} else {
		e, err = c.GetRemote(name)
	}

	if err != nil {
		return err
	}

	sps, err := e.GetAllServices()
	if err != nil {
		return fmt.Errorf("while retrieving services: %s", err)
	}

	ch := make(chan *status)
	for name, sp := range sps {
		name := name
		for _, service := range sp {
			service := service
			go func() {
				ch <- doStatusCheck(name, service)
			}()
		}
	}

	// map storing statuses by name
	smap := make(map[string]*status)
	for _, sp := range sps {
		for range sp {
			s := <-ch
			if s == nil {
				continue
			}
			smap[s.name] = s
		}
	}

	// list in alphanumeric order
	names := make([]string, 0, len(smap))
	for n := range smap {
		names = append(names, n)
	}
	sort.Strings(names)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, statusLine, "SERVICE", "STATUS", "VERSION", "URI")
	for _, n := range names {
		s := smap[n]
		fmt.Fprintf(tw, statusLine, strings.Title(s.name), s.status, s.version, s.uri)
	}
	tw.Flush()

	return doTokenCheck(e)
}

func doStatusCheck(name string, sp endpoint.Service) *status {
	uri := sp.URI()
	version, err := sp.Status()
	if err != nil {
		if err == endpoint.ErrStatusNotSupported {
			return nil
		}
		return &status{name: name, uri: uri, status: "N/A"}
	}
	return &status{name: name, uri: uri, status: "OK", version: version}
}

func doTokenCheck(e *endpoint.Config) error {
	if e.Token == "" {
		fmt.Println("\nNo authentication token set (logged out).")
		return nil
	}
	if err := e.VerifyToken(""); err != nil {
		fmt.Println("\nAuthentication token is invalid (please login again).")
		return err

	}
	fmt.Println("\nValid authentication token set (logged in).")
	return nil
}
