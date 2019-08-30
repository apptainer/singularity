// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/sylabs/singularity/e2e/internal/e2e"
)

const instanceStartPort = 11372

type instance struct {
	Image    string `json:"img"`
	Instance string `json:"instance"`
	Pid      int    `json:"pid"`
}

type instanceList struct {
	Instances []instance `json:"instances"`
}

func (c *ctx) listInstance(t *testing.T, listArgs ...string) (stdout string, stderr string, success bool) {
	var args []string

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance list"),
		e2e.WithArgs(args...),
		e2e.PostRun(func(t *testing.T) {
			success = !t.Failed()
		}),
		e2e.ExpectExit(0, e2e.GetStreams(&stdout, &stderr)),
	)

	return
}

func (c *ctx) stopInstance(t *testing.T, instance string, stopArgs ...string) (stdout string, stderr string, success bool) {
	args := stopArgs

	if instance != "" {
		args = append(args, instance)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance stop"),
		e2e.WithArgs(args...),
		e2e.PostRun(func(t *testing.T) {
			success = !t.Failed()
		}),
		e2e.ExpectExit(0, e2e.GetStreams(&stdout, &stderr)),
	)

	return
}

func (c *ctx) execInstance(t *testing.T, instance string, execArgs ...string) (stdout string, stderr string, success bool) {
	args := []string{"instance://" + instance}
	args = append(args, execArgs...)

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("exec"),
		e2e.WithArgs(args...),
		e2e.PostRun(func(t *testing.T) {
			success = !t.Failed()
		}),
		e2e.ExpectExit(0, e2e.GetStreams(&stdout, &stderr)),
	)

	return
}

// Return the number of currently running instances.
func (c *ctx) expectedNumberOfInstances(t *testing.T, n int) {
	nbInstances := -1

	listInstancesFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
		var instances instanceList

		if err := json.Unmarshal([]byte(r.Stdout), &instances); err != nil {
			t.Errorf("Error while decoding JSON from 'instance list': %v", err)
		}
		nbInstances = len(instances.Instances)
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance list"),
		e2e.WithArgs([]string{"--json"}...),
		e2e.PostRun(func(t *testing.T) {
			if !t.Failed() && n != nbInstances {
				t.Errorf("%d instance(s) are running, was expecting %d", nbInstances, n)
			}
		}),
		e2e.ExpectExit(0, listInstancesFn),
	)
}

// Sends a deterministic message to an echo server and expects the same message
// in response.
func echo(t *testing.T, port int) {
	const message = "b40cbeaaea293f7e8bd40fb61f389cfca9823467\n"

	sock, sockErr := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if sockErr != nil {
		t.Errorf("Failed to dial echo server: %v", sockErr)
		return
	}

	fmt.Fprintf(sock, message)

	response, responseErr := bufio.NewReader(sock).ReadString('\n')
	if responseErr != nil || response != message {
		t.Errorf("Bad response: err = %v, response = %v", responseErr, response)
	}
}
