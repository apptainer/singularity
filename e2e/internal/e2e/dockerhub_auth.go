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

<<<<<<< HEAD
	"github.com/hpcng/singularity/internal/pkg/util/user"
	"github.com/hpcng/singularity/pkg/syfs"
	auth "github.com/oras-project/oras-go/pkg/auth/docker"
=======
	auth "github.com/oras-project/oras-go/pkg/auth/docker"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"github.com/sylabs/singularity/pkg/syfs"
>>>>>>> sylabs41-2
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
