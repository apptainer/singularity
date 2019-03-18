// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// +build seccomp

package seccomp

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	"github.com/opencontainers/runtime-tools/generate"

	"github.com/sylabs/singularity/internal/pkg/sylog"

	cseccomp "github.com/kubernetes-sigs/cri-o/pkg/seccomp"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	lseccomp "github.com/seccomp/libseccomp-golang"
)

var scmpArchMap = map[specs.Arch]lseccomp.ScmpArch{
	"":                    lseccomp.ArchNative,
	specs.ArchX86:         lseccomp.ArchX86,
	specs.ArchX86_64:      lseccomp.ArchAMD64,
	specs.ArchX32:         lseccomp.ArchX32,
	specs.ArchARM:         lseccomp.ArchARM,
	specs.ArchAARCH64:     lseccomp.ArchARM64,
	specs.ArchMIPS:        lseccomp.ArchMIPS,
	specs.ArchMIPS64:      lseccomp.ArchMIPS64,
	specs.ArchMIPS64N32:   lseccomp.ArchMIPS64N32,
	specs.ArchMIPSEL:      lseccomp.ArchMIPSEL,
	specs.ArchMIPSEL64:    lseccomp.ArchMIPSEL64,
	specs.ArchMIPSEL64N32: lseccomp.ArchMIPSEL64N32,
	specs.ArchPPC:         lseccomp.ArchPPC,
	specs.ArchPPC64:       lseccomp.ArchPPC64,
	specs.ArchPPC64LE:     lseccomp.ArchPPC64LE,
	specs.ArchS390:        lseccomp.ArchS390,
	specs.ArchS390X:       lseccomp.ArchS390X,
}

var scmpActionMap = map[specs.LinuxSeccompAction]lseccomp.ScmpAction{
	specs.ActKill:  lseccomp.ActKill,
	specs.ActTrap:  lseccomp.ActTrap,
	specs.ActErrno: lseccomp.ActErrno,
	specs.ActTrace: lseccomp.ActTrace,
	specs.ActAllow: lseccomp.ActAllow,
}

var scmpCompareOpMap = map[specs.LinuxSeccompOperator]lseccomp.ScmpCompareOp{
	specs.OpNotEqual:     lseccomp.CompareNotEqual,
	specs.OpLessThan:     lseccomp.CompareLess,
	specs.OpLessEqual:    lseccomp.CompareLessOrEqual,
	specs.OpEqualTo:      lseccomp.CompareEqual,
	specs.OpGreaterEqual: lseccomp.CompareGreaterEqual,
	specs.OpGreaterThan:  lseccomp.CompareGreater,
	specs.OpMaskedEqual:  lseccomp.CompareMaskedEqual,
}

func prctl(option uintptr, arg2 uintptr, arg3 uintptr, arg4 uintptr, arg5 uintptr) syscall.Errno {
	_, _, err := syscall.Syscall6(syscall.SYS_PRCTL, option, arg2, arg3, arg4, arg5, 0)
	return err
}

func hasConditionSupport() bool {
	major, minor, micro := lseccomp.GetLibraryVersion()
	return (major > 2) || (major == 2 && minor >= 2) || (major == 2 && minor == 2 && micro >= 1)
}

// Enabled returns wether seccomp is enabled or not
func Enabled() bool {
	return true
}

// LoadSeccompConfig loads seccomp configuration filter for the current process
func LoadSeccompConfig(config *specs.LinuxSeccomp, noNewPrivs bool) error {
	if err := prctl(syscall.PR_GET_SECCOMP, 0, 0, 0, 0); err == syscall.EINVAL {
		return fmt.Errorf("can't load seccomp filter: not supported by kernel")
	}

	if err := prctl(syscall.PR_SET_SECCOMP, 2, 0, 0, 0); err == syscall.EINVAL {
		return fmt.Errorf("can't load seccomp filter: SECCOMP_MODE_FILTER not supported")
	}

	if config == nil {
		return fmt.Errorf("empty config passed")
	}

	if len(config.DefaultAction) == 0 {
		return fmt.Errorf("a defaultAction must be provided")
	}

	supportCondition := hasConditionSupport()
	if supportCondition == false {
		sylog.Warningf("seccomp rule conditions are not supported with libseccomp under 2.2.1")
	}

	scmpAction, ok := scmpActionMap[config.DefaultAction]
	if !ok {
		return fmt.Errorf("invalid action '%s' specified", config.DefaultAction)
	}
	if scmpAction == lseccomp.ActErrno {
		scmpAction = scmpAction.SetReturnCode(1)
	}

	filter, err := lseccomp.NewFilter(scmpAction)
	if err != nil {
		return fmt.Errorf("error creating new filter: %s", err)
	}

	if err := filter.SetNoNewPrivsBit(noNewPrivs); err != nil {
		return fmt.Errorf("failed to set no new priv flag: %s", err)
	}

	for _, arch := range config.Architectures {
		scmpArch, ok := scmpArchMap[arch]
		if !ok {
			return fmt.Errorf("invalid architecture '%s' specified", arch)
		}

		if err := filter.AddArch(scmpArch); err != nil {
			return fmt.Errorf("error adding architecture: %s", err)
		}
	}

	for _, syscall := range config.Syscalls {
		if len(syscall.Names) == 0 {
			return fmt.Errorf("no syscall specified for the rule")
		}

		scmpAction, ok = scmpActionMap[syscall.Action]
		if !ok {
			return fmt.Errorf("invalid action '%s' specified", syscall.Action)
		}
		if scmpAction == lseccomp.ActErrno {
			scmpAction = scmpAction.SetReturnCode(1)
		}

		for _, sysName := range syscall.Names {
			sysNr, err := lseccomp.GetSyscallFromName(sysName)
			if err != nil {
				continue
			}

			if len(syscall.Args) == 0 || supportCondition == false {
				if err := filter.AddRule(sysNr, scmpAction); err != nil {
					return fmt.Errorf("failed adding seccomp rule for syscall %s: %s", sysName, err)
				}
			} else {
				conditions, err := addSyscallRuleContitions(syscall.Args)
				if err != nil {
					return err
				}
				if err := filter.AddRuleConditional(sysNr, scmpAction, conditions); err != nil {
					return fmt.Errorf("failed adding rule condition for syscall %s: %s", sysName, err)
				}
			}
		}
	}

	if err = filter.Load(); err != nil {
		return fmt.Errorf("failed loading seccomp filter: %s", err)
	}

	return nil
}

func addSyscallRuleContitions(args []specs.LinuxSeccompArg) ([]lseccomp.ScmpCondition, error) {
	var maxIndex uint = 6
	conditions := make([]lseccomp.ScmpCondition, 0)

	for _, arg := range args {
		if arg.Index >= maxIndex {
			return conditions, fmt.Errorf("the maximum index of syscall arguments is %d: given %d", maxIndex, arg.Index)
		}
		operator, ok := scmpCompareOpMap[arg.Op]
		if !ok {
			return conditions, fmt.Errorf("invalid operator encountered %s", arg.Op)
		}
		cond, err := lseccomp.MakeCondition(arg.Index, operator, arg.Value, arg.ValueTwo)
		if err != nil {
			return conditions, fmt.Errorf("error making syscall rule condition: %s", err)
		}
		conditions = append(conditions, cond)
	}

	return conditions, nil
}

// LoadProfileFromFile loads seccomp rules from json file and fill in
// provided OCI configuration
func LoadProfileFromFile(profile string, generator *generate.Generator) error {
	file, err := os.Open(profile)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	if generator.Config.Linux == nil {
		generator.Config.Linux = &specs.Linux{}
	}
	if generator.Config.Linux.Seccomp == nil {
		generator.Config.Linux.Seccomp = &specs.LinuxSeccomp{}
	}
	if generator.Config.Process == nil {
		generator.Config.Process = &specs.Process{}
	}
	if generator.Config.Process.Capabilities == nil {
		generator.Config.Process.Capabilities = &specs.LinuxCapabilities{}
	}
	if err := cseccomp.LoadProfileFromBytes(data, generator); err != nil {
		return err
	}
	return nil
}
