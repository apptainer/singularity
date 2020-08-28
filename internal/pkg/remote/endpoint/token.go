// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package endpoint

import (
	"fmt"
	"net/http"

	"github.com/sylabs/singularity/internal/pkg/remote/credential"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/sylog"
	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
)

// VerifyToken returns an error if a token is not valid
func (ep *Config) VerifyToken() (err error) {
	if ep.URI == "" {
		return fmt.Errorf("no endpoint URI")
	}

	defer func() {
		if err == nil {
			sylog.Infof("API Key Verified!")
		}
	}()

	if ep.Token == "" {
		fmt.Printf("Generate an API Key at https://%s/auth/tokens, and paste here:\n", ep.URI)
		ep.Token, err = interactive.AskQuestionNoEcho("API Key: ")
		if err != nil {
			return err
		}
	}

	sp, err := ep.GetAllServices()
	if err != nil {
		return err
	}

	ts, ok := sp[Token]
	if !ok || len(ts) == 0 {
		return fmt.Errorf("no authentication service found")
	}

	client := &http.Client{
		Timeout: defaultTimeout,
	}
	req, err := http.NewRequest(http.MethodGet, ts[0].URI()+"/v1/token-status", nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", credential.TokenPrefix+ep.Token)
	req.Header.Set("User-Agent", useragent.Value())

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request to server: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		convStatus, ok := errorCodeMap[res.StatusCode]
		if !ok {
			convStatus = "Unknown"
		}
		return fmt.Errorf("error response from server: %v", convStatus)
	}

	return nil
}
