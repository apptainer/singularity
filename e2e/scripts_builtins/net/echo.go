package net

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// net-echo builtin
// usage:
// net-echo tcp|udp ip:port
func netEcho(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("net-echo requires 2 arguments")
	}
	b, err := ioutil.ReadAll(mc.Stdin)
	if err != nil {
		return fmt.Errorf("error while reading input message: %s", err)
	}
	message := string(b)

	proto := args[0]
	switch proto {
	case "tcp", "udp":
	default:
		return fmt.Errorf("protocol %s not supported", proto)
	}

	sock, err := net.Dial(proto, args[1])
	if err != nil {
		return fmt.Errorf("failed to reach %s: %s", args[1], err)
	}
	defer sock.Close()

	if _, err := fmt.Fprintf(sock, message); err != nil {
		return fmt.Errorf("failed to write message: %s", err)
	}

	response, err := bufio.NewReader(sock).ReadString('\n')
	if err != nil {
		return fmt.Errorf("error while getting response %s", err)
	}
	if _, err := fmt.Fprintf(mc.Stdout, "%s\n", response); err != nil {
		return fmt.Errorf("output error: %s", err)
	}

	return nil
}

func init() {
	stest.RegisterCommandBuiltin("net-echo", netEcho)
}
