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
	"time"

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

	c.expectInstance(t, instance, 0)

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

// Check if there is the number of expected instances with the provided name.
func (c *ctx) expectInstance(t *testing.T, name string, nb int) {
	listInstancesFn := func(t *testing.T, r *e2e.SingularityCmdResult) {
		var instances instanceList

		if err := json.Unmarshal([]byte(r.Stdout), &instances); err != nil {
			t.Errorf("Error while decoding JSON from 'instance list': %v", err)
		}
		if nb != len(instances.Instances) {
			t.Errorf("%d instance %q found, expected %d", len(instances.Instances), name, nb)
		}
	}

	c.env.RunSingularity(
		t,
		e2e.WithProfile(c.profile),
		e2e.WithCommand("instance list"),
		e2e.WithArgs([]string{"--json", name}...),
		e2e.ExpectExit(0, listInstancesFn),
	)
}

// Sends a deterministic message to an echo server and expects the same message
// in response.
func echo(t *testing.T, port int) {
	const message = "b40cbeaaea293f7e8bd40fb61f389cfca9823467\n"

	// give it some time for responding, attempt 10 times by
	// waiting 100 millisecond between each try
	for retries := 0; ; retries++ {
		sock, sockErr := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(port))
		if sockErr != nil && retries < 10 {
			time.Sleep(100 * time.Millisecond)
			continue
		} else if sockErr != nil {
			t.Errorf("Failed to dial echo server: %v", sockErr)
			return
		}

		fmt.Fprintf(sock, message)

		response, responseErr := bufio.NewReader(sock).ReadString('\n')
		if responseErr != nil || response != message {
			t.Errorf("Bad response: err = %v, response = %v", responseErr, response)
		}
		break
	}
}
