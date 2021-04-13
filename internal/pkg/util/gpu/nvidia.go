// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package gpu

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/hpcng/singularity/internal/pkg/util/bin"
	"github.com/hpcng/singularity/internal/pkg/util/env"
	"github.com/hpcng/singularity/internal/pkg/util/fs"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/hpcng/singularity/pkg/util/capabilities"
	"github.com/hpcng/singularity/pkg/util/slice"
)

var ErrNvCCLIInsecure = errors.New("nvidia-container-cli is not owned by root user")

// nVDriverCapabilities is the set of driver capabilities supported by nvidia-container-cli.
// See: https://github.com/nvidia/nvidia-container-runtime#nvidia_driver_capabilities
var nVDriverCapabilities = []string{
	"compute",
	"compat32",
	"graphics",
	"utility",
	"video",
	"display",
}

// nVDriverDefaultCapabilities is the default set of nvidia-container-cli driver capabilities.
// It is used if NVIDIA_DRIVER_CAPABILITIES is not set.
// See: https://github.com/nvidia/nvidia-container-runtime#nvidia_driver_capabilities
var nVDriverDefaultCapabilities = []string{
	"compute",
	"utility",
}

// nVCLIAmbientCaps is the ambient capability set required by nvidia-container-cli.
var nVCLIAmbientCaps = []uintptr{
	uintptr(capabilities.Map["CAP_KILL"].Value),
	uintptr(capabilities.Map["CAP_SETUID"].Value),
	uintptr(capabilities.Map["CAP_SETGID"].Value),
	uintptr(capabilities.Map["CAP_SYS_CHROOT"].Value),
	uintptr(capabilities.Map["CAP_CHOWN"].Value),
	uintptr(capabilities.Map["CAP_FOWNER"].Value),
	uintptr(capabilities.Map["CAP_MKNOD"].Value),
	uintptr(capabilities.Map["CAP_SYS_ADMIN"].Value),
	uintptr(capabilities.Map["CAP_DAC_READ_SEARCH"].Value),
	uintptr(capabilities.Map["CAP_SYS_PTRACE"].Value),
	uintptr(capabilities.Map["CAP_DAC_OVERRIDE"].Value),
	uintptr(capabilities.Map["CAP_SETPCAP"].Value),
}

// GetNvCCLIPath finds the path to nvidia-container-cli.
// Returns ErrNvCCLIInsecure if it is not owned by root.
func GetNvCCLIPath() (path string, err error) {
	path, err = bin.FindBin("nvidia-container-cli")
	if err != nil {
		return "", err
	}

	// The nvidia-container-cli binary must be owned by root, as it is called with broad
	// capabilities, and as root in the setuid flow.
	if !fs.IsOwner(path, 0) {
		return "", ErrNvCCLIInsecure
	}

	return path, nil
}

// NVCLIConfigure calls out to the nvidia-container-cli configure operation.
// This sets up the GPU with the container. Note that the ability to set a fairly broad set of
// ambient capabilities is required. This function will error if the bounding set does not include
// NvidiaContainerCLIAmbientCaps.
func NVCLIConfigure(nvCCLIPath string, flags []string, rootfs string, runAsRoot bool) error {
	nccArgs := []string{"configure"}

	// If we will not run as root (i.e. we are in a user namespace), specify --user as a global
	// flag, or nvidia-container-cli will fail.
	if !runAsRoot {
		nccArgs = []string{"--user", "configure"}
	}

	nccArgs = append(nccArgs, flags...)
	nccArgs = append(nccArgs, rootfs)

	sylog.Debugf("nvidia-container-cli binary: %q args: %q", nvCCLIPath, nccArgs)

	cmd := exec.Command(nvCCLIPath, nccArgs...)
	cmd.Env = os.Environ()
	// We are called from the RPC server which has an empty PATH.
	// nvidia-container-cli requires a default sensible PATH to work correctly.
	cmd.Env = append(cmd.Env, "PATH="+env.DefaultPath)

	// We need to run nvidia-container-cli as root when we are in the setuid flow
	// without a user namespace in play.
	if runAsRoot {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{Uid: 0, Gid: 0},
		}
	} else {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.AmbientCaps = nVCLIAmbientCaps
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nvidia-container-cli failed with %v: %s", err, stdoutStderr)
	}
	return nil
}

// NVCLIEnvToFlags reads the environment variables supported by nvidia-container-runtime
// and converts them to flags for nvidia-container-cli.
// See: https://github.com/nvidia/nvidia-container-runtime#environment-variables-oci-spec
func NVCLIEnvToFlags() (flags []string, err error) {
	// We don't support cgroups related usage yet.
	flags = []string{"--no-cgroups"}

	ldConfig, err := bin.FindBin("ldconfig")
	if err != nil {
		return nil, fmt.Errorf("could not lookup ldconfig: %v", err)
	}
	flags = append(flags, "--ldconfig=@"+ldConfig)

	if val := os.Getenv("NVIDIA_VISIBLE_DEVICES"); val != "" {
		flags = append(flags, "--device="+val)
	}

	if val := os.Getenv("NVIDIA_MIG_CONFIG_DEVICES"); val != "" {
		flags = append(flags, "--mig-config="+val)
	}

	if val := os.Getenv("NVIDIA_MIG_MONITOR_DEVICES"); val != "" {
		flags = append(flags, "--mig-monitor="+val)
	}

	// Driver capabilities have a default, but can be overridden.
	caps := nVDriverDefaultCapabilities
	if val := os.Getenv("NVIDIA_DRIVER_CAPABILITIES"); val != "" {
		caps = strings.Split(val, ",")
	}

	for _, cap := range caps {
		if slice.ContainsString(nVDriverCapabilities, cap) {
			flags = append(flags, "--"+cap)
		} else {
			return nil, fmt.Errorf("unknown NVIDIA_DRIVER_CAPABILITIES value: %s", cap)
		}
	}

	// One --require flag for each NVIDIA_REQUIRE_* environment
	// https://github.com/nvidia/nvidia-container-runtime#nvidia_require_
	if val := os.Getenv("NVIDIA_DISABLE_REQUIRE"); val == "" {
		for _, e := range os.Environ() {
			if strings.HasPrefix(e, "NVIDIA_REQUIRE_") {
				req := strings.SplitN(e, "=", 2)[1]
				flags = append(flags, "--require="+req)
			}
		}
	}

	return flags, nil
}
