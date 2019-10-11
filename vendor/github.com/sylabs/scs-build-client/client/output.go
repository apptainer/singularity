// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
)

// OutputReader interface is used to read the websocket output from the stream
type OutputReader interface {
	// Read is called when a websocket message is received
	Read(messageType int, p []byte) (int, error)
}

// GetOutput reads the build output log for the provided buildID - streaming to
// OutputReader. The context controls the lifetime of the request.
func (c *Client) GetOutput(ctx context.Context, buildID string, or OutputReader) error {
	wsScheme := "ws"
	if c.BaseURL.Scheme == "https" {
		wsScheme = "wss"
	}
	u := c.BaseURL.ResolveReference(&url.URL{
		Scheme:   wsScheme,
		Host:     c.BaseURL.Host,
		Path:     "/v1/build-ws/" + buildID,
		RawQuery: "",
	})

	h := http.Header{}
	c.setRequestHeaders(h)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ws, resp, err := websocket.DefaultDialer.DialContext(ctx, u.String(), h)
	if err != nil {
		c.Logger.Logf("websocket dial err - %s, partial response: %+v", err, resp)
		return err
	}
	defer resp.Body.Close()
	defer ws.Close()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		fmt.Printf("\rShutting down due to signal: %v\n", <-sigCh)

		if err := c.Cancel(ctx, buildID); err != nil {
			c.Logger.Logf("build cancellation request failed: %v", err)
		}

		cancel()

	}()

	for {
		// Read from websocket
		mt, msg, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return nil
			}
			c.Logger.Logf("websocket read message err - %s", err)
			return err
		}

		n, err := or.Read(mt, msg)
		if err != nil {
			return err
		}
		if n != len(msg) {
			return fmt.Errorf("did not read all message contents: %d != %d", n, len(msg))
		}

	}
}
