// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

//go:build seccomp
// +build seccomp

package seccomp

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"

	cseccomp "github.com/containers/common/pkg/seccomp"
	"github.com/hpcng/singularity/internal/pkg/runtime/engine/config/oci/generate"
	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/opencontainers/runtime-spec/specs-go"
	lseccomp "github.com/seccomp/libseccomp-golang"
)

var (
	ErrInvalidAction    = errors.New("invalid action")
	ErrUnsupportedErrno = errors.New("errno is not supported for action")
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

// getDefaultErrno returns the default errNo that is specified by the specs
//
// https://github.com/opencontainers/runtime-spec/blob/main/config-linux.md#seccomp
//
// defaultErrnoRet (uint, OPTIONAL) - the errno return code to use. Some actions
// like SCMP_ACT_ERRNO and SCMP_ACT_TRACE allow to specify the errno code to
// return. When the action doesn't support an errno, the runtime MUST print and
// error and fail. If not specified then its default value is EPERM
//
func getDefaultErrno(config *specs.LinuxSeccomp) (errnoRet *uint, err error) {
	// If there is no attempt to explicitly set a defaultErrnoRet then a default
	// or explicit ERRNO/TRACE action should return EPERM.
	if config.DefaultErrnoRet == nil {
		eperm := uint(syscall.EPERM)
		return &eperm, nil
	}

	// defaultErrno is set with a defaultAction of ERRNO or TRACE
	if config.DefaultAction == specs.ActErrno || config.DefaultAction == specs.ActTrace {
		return config.DefaultErrnoRet, nil
	}

	// defaultErrno is set for a defaultAction that doesn't support it
	return nil, fmt.Errorf("defaultAction: %w", ErrUnsupportedErrno)
}

// getAction returns the approriate libseccomp action for a given containers/common seccomp action
func getAction(specAction specs.LinuxSeccompAction, errnoRet *uint, defaultErrNoRet uint) (scmpAction lseccomp.ScmpAction, err error) {
	scmpAction, ok := scmpActionMap[specAction]
	if !ok {
		return lseccomp.ActInvalid, fmt.Errorf("%v: %w", specAction, ErrInvalidAction)
	}

	// Errno or Trace must set an Errno
	if specAction == specs.ActErrno || specAction == specs.ActTrace {
		// errnoRet override of the default
		if errnoRet != nil {
			return scmpAction.SetReturnCode(int16(*errnoRet)), nil
		}
		// defaultErrnoRet (which is EPERM if not specified)
		return scmpAction.SetReturnCode(int16(defaultErrNoRet)), nil
	}

	// Other actions which don't take an errno
	if errnoRet != nil {
		return lseccomp.ActInvalid, fmt.Errorf("%v, %w", specAction, ErrUnsupportedErrno)
	}

	return scmpAction, nil
}

// Enabled returns whether seccomp is enabled.
func Enabled() bool {
	return true
}

// LoadSeccompConfig loads seccomp configuration filter for the current process.
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
	if !supportCondition {
		sylog.Warningf("seccomp rule conditions are not supported with libseccomp under 2.2.1")
	}

	defaultErrno, err := getDefaultErrno(config)
	if err != nil {
		return fmt.Errorf("can't load default action: %w", err)
	}
	if defaultErrno == nil {
		return fmt.Errorf("internal error - computed defaultErrno cannot be nil")
	}

	defaultAction, err := getAction(config.DefaultAction, nil, *defaultErrno)
	if err != nil {
		return fmt.Errorf("can't load default action: %w", err)
	}

	filter, err := lseccomp.NewFilter(defaultAction)
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

		scmpAction, err := getAction(syscall.Action, syscall.ErrnoRet, *defaultErrno)
		if err != nil {
			return fmt.Errorf("error adding action: %w", err)
		}

		// If the action is equal to the default action we skip the rule
		// silently, as it is redundant and AddRule will error inserting it.
		if scmpAction == defaultAction {
			sylog.Debugf("Skipping redundant seccomp rule for %v %v", syscall.Names, scmpAction)
			continue
		}

		for _, sysName := range syscall.Names {
			sysNr, err := lseccomp.GetSyscallFromName(sysName)
			if err != nil {
				continue
			}

			if len(syscall.Args) == 0 || !supportCondition {
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

// LoadProfileFromFile loads seccomp rules from json file and fill in provided OCI configuration.
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
	if generator.Config.Process == nil {
		generator.Config.Process = &specs.Process{}
	}
	if generator.Config.Process.Capabilities == nil {
		generator.Config.Process.Capabilities = &specs.LinuxCapabilities{}
	}

	seccompConfig, err := cseccomp.LoadProfileFromBytes(data, generator.Config)
	if err != nil {
		return err
	}
	generator.Config.Linux.Seccomp = seccompConfig

	return nil
}
