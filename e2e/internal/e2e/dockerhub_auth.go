// Copyright (c) 2020, Control Command Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/syfs"
)

const dockerHub = "docker.io"

func SetupDockerHubCredentials(t *testing.T) {
	var unprivUser, privUser *user.User

	username := os.Getenv("E2E_DOCKER_USERNAME")
	pass := os.Getenv("E2E_DOCKER_PASSWORD")

	if username == "" && pass == "" {
		t.Log("No DockerHub credentials supplied, DockerHub rate limits could be hit")
		return
	}

	unprivUser = CurrentUser(t)
	writeDockerHubCredentials(t, unprivUser.Dir, username, pass)
	Privileged(func(t *testing.T) {
		privUser = CurrentUser(t)
		writeDockerHubCredentials(t, privUser.Dir, username, pass)
	})(t)
}

func writeDockerHubCredentials(t *testing.T, dir, username, pass string) {
	configPath := filepath.Join(dir, ".singularity", syfs.DockerConfFile)
	cli, err := auth.NewClient(configPath)
	if err != nil {
		t.Fatalf("failed to get docker auth client: %v", err)
	}
	if err := cli.Login(context.Background(), dockerHub, username, pass, false); err != nil {
		t.Fatalf("failed to login to dockerhub: %v", err)
	}
}
