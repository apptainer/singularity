// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package cli

import (
	"fmt"
)

func getHypervisorArgs(sifImage, bzImage, initramfs, singAction, cliExtra string) []string {
	// Setup some needed variables
	appendArgs := fmt.Sprintf("root=/dev/ram0 console=ttyS0 quiet singularity_action=%s singularity_arguments=\"%s\"", singAction, cliExtra)

	args := []string{"/usr/libexec/qemu-kvm", "-cpu", "host", "-smp", VMCPU, "-enable-kvm", "-device", "virtio-rng-pci", "-display", "none", "-realtime", "mlock=on", "-hda", sifImage, "-serial", "stdio", "-kernel", bzImage, "-initrd", initramfs, "-m", VMRAM, "-append", appendArgs}

	return args
}
