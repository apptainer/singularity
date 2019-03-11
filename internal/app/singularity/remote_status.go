// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularity

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/singularity/internal/pkg/remote"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

const statusLine = "%s\t%s\t%s\n"

type status struct {
	name    string
	uri     string
	status  string
	version string
}

type scsAssets map[string]string

// RemoteStatus checks status of services related to an endpoint
func RemoteStatus(configFile, name string) (err error) {
	c := &remote.Config{}
	file, err := os.OpenFile(configFile, os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("No Remote configurations")
		}
		return fmt.Errorf("while opening remote config file: %s", err)
	}
	defer file.Close()

	c, err = remote.ReadFrom(file)
	if err != nil {
		return fmt.Errorf("while parsing capability config data: %s", err)
	}

	e, err := c.GetRemote(name)
	if err != nil {
		return err
	}

	a, err := e.GetAllServiceURIs()
	if err != nil {
		return fmt.Errorf("while getting asset configuration: %s", err)
	}

	ch := make(chan status)
	for name, uri := range a {
		go doStatusCheck(name, uri, ch)
	}

	// map storing statuses by name
	smap := make(map[string]status)
	for range a {
		s := <-ch
		smap[s.name] = s
	}

	// list in alphanumeric order
	var names []string
	for n := range smap {
		names = append(names, n)
	}
	sort.Strings(names)

	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, statusLine, "SERVICE", "STATUS", "VERSION")
	for _, n := range names {
		s := smap[n]
		fmt.Fprintf(tw, statusLine, strings.Title(s.name+" Service"), s.status, s.version)
	}
	tw.Flush()

	return nil
}

// VersionResponse - Response form the API for a version request
type VersionResponse struct {
	Version string `json:"version"`
}

func getStatus(url string) (version string, err error) {
	client := &http.Client{
		Timeout: (30 * time.Second),
	}

	req, err := http.NewRequest(http.MethodGet, url+"/version", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request to server:\n\t%v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error response from server: %v", res.StatusCode)
	}

	var vRes VersionResponse
	if err := jsonresp.ReadResponse(res.Body, &vRes); err != nil {
		return "", err
	}

	return vRes.Version, nil
}

func doStatusCheck(name, uri string, ch chan<- status) {
	stat, err := getStatus(uri)
	if err != nil {
		ch <- status{name: name, uri: uri, status: "N/A"}
		return
	}
	ch <- status{name: name, uri: uri, status: "OK", version: stat}
}
