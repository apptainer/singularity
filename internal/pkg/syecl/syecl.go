// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package syecl implements the loading and management of the container
// execution control list feature. This code uses the TOML config file standard
// to extract the structured configuration for activating or disabling the list
// and for the implementation of the execution groups.
package syecl

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml"
	"github.com/sylabs/singularity/pkg/signing"
)

// EclConfig describes the structure of an execution control list configuration file
type EclConfig struct {
	Activated  bool        `toml:"activated"` // toggle the activation of the ECL rules
	ExecGroups []execgroup `toml:"execgroup"` // Slice of all execution groups
}

// execgroup describes an execution group, the main unit of configuration:
//	TagName: a descriptive identifier
//	ListMode: whether the execgroup follows a whitelist, whitestrict or blacklist model
//		whitelist: one or more KeyFP's present and verified,
//		whitestrict: all KeyFP's present and verified,
//		blacklist: none of the KeyFP should be present
//	DirPath: containers must be stored in this directory path
//	KeyFPs: list of Key Fingerprints of entities to verify
type execgroup struct {
	TagName  string   `toml:"tagname"`
	ListMode string   `toml:"mode"`
	DirPath  string   `toml:"dirpath"`
	KeyFPs   []string `toml:"keyfp"`
}

// LoadConfig opens an ECL config file and unmarshals it into structures
func LoadConfig(confPath string) (ecl EclConfig, err error) {
	// read in the ECL config file
	b, err := ioutil.ReadFile(confPath)
	if err != nil {
		return
	}

	// Unmarshal config file
	err = toml.Unmarshal(b, &ecl)
	return
}

// PutConfig takes the content of an EclConfig struct and Marshals it to file
func PutConfig(ecl EclConfig, confPath string) (err error) {
	data, err := toml.Marshal(ecl)
	if err != nil {
		return
	}

	return ioutil.WriteFile(confPath, data, 0600)
}

// ValidateConfig makes sure paths from configs are fully resolved and that
// values from an execgroup are logically correct.
func (ecl *EclConfig) ValidateConfig() (err error) {
	m := map[string]bool{}

	for _, v := range ecl.ExecGroups {
		if m[v.DirPath] {
			return fmt.Errorf("a specific dirpath can only appear in one execgroup: %s", v.DirPath)
		}
		m[v.DirPath] = true

		// if we allow containers everywhere, don't test dirpath constraint
		if v.DirPath != "" {
			path, err := filepath.EvalSymlinks(v.DirPath)
			if err != nil {
				return err
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if v.DirPath != abs {
				return fmt.Errorf("all execgroup dirpath`s should be fully cleaned with symlinks resolved")
			}
		}
		if v.ListMode != "whitelist" && v.ListMode != "whitestrict" && v.ListMode != "blacklist" {
			return fmt.Errorf("the mode field can only be either: whitelist, whitestrict, blacklist")
		}
		for _, k := range v.KeyFPs {
			decoded, err := hex.DecodeString(k)
			if err != nil || len(decoded) != 20 {
				return fmt.Errorf("expecting a 40 chars hex fingerprint string")
			}
		}
	}
	return
}

// checkWhiteList evaluates authorization by requiring at least 1 entity
func checkWhiteList(fp *os.File, egroup *execgroup) (ok bool, err error) {
	// get all signing entities fingerprints on the primary partition
	keyfps, err := signing.GetSignEntitiesFp(fp)
	if err != nil {
		return
	}
	// was the primary partition signed by an authorized entity?
	for _, v := range egroup.KeyFPs {
		for _, u := range keyfps {
			if v == u {
				ok = true
			}
		}
	}
	if !ok {
		return false, fmt.Errorf("%s is not signed by required entities", fp.Name())
	}

	return true, nil
}

// checkWhiteStrict evaluates authorization by requiring all entities
func checkWhiteStrict(fp *os.File, egroup *execgroup) (ok bool, err error) {
	// get all signing entities fingerprints on the primary partition
	keyfps, err := signing.GetSignEntitiesFp(fp)
	if err != nil {
		return
	}

	// was the primary partition signed by all authorized entity?
	m := map[string]bool{}
	for _, v := range egroup.KeyFPs {
		m[v] = false
		for _, u := range keyfps {
			if v == u {
				m[v] = true
			}
		}
	}
	for _, v := range m {
		if !v {
			return false, fmt.Errorf("%s is not signed by required entities", fp.Name())
		}
	}

	return true, nil
}

// checkBlackList evaluates authorization by requiring all entities to be absent
func checkBlackList(fp *os.File, egroup *execgroup) (ok bool, err error) {
	// get all signing entities fingerprints on the primary partition
	keyfps, err := signing.GetSignEntitiesFp(fp)
	if err != nil {
		return
	}
	// was the primary partition signed by an authorized entity?
	for _, v := range egroup.KeyFPs {
		for _, u := range keyfps {
			if v == u {
				return false, fmt.Errorf("%s is signed by a forbidden entity", fp.Name())
			}
		}
	}

	return true, nil
}

func shouldRun(ecl *EclConfig, fp *os.File) (ok bool, err error) {
	var egroup *execgroup

	// look what execgroup a container is part of
	for _, v := range ecl.ExecGroups {
		if filepath.Dir(fp.Name()) == v.DirPath {
			egroup = &v
			break
		}
	}
	// go back at it and this time look for an empty dirpath execgroup to fallback into
	if egroup == nil {
		for _, v := range ecl.ExecGroups {
			if v.DirPath == "" {
				egroup = &v
				break
			}
		}
	}

	if egroup == nil {
		return false, fmt.Errorf("%s not part of any execgroup", fp.Name())
	}

	switch egroup.ListMode {
	case "whitelist":
		return checkWhiteList(fp, egroup)
	case "whitestrict":
		return checkWhiteStrict(fp, egroup)
	case "blacklist":
		return checkBlackList(fp, egroup)
	}

	return false, fmt.Errorf("ecl config file invalid")
}

// ShouldRun determines if a container should run according to its execgroup rules
func (ecl *EclConfig) ShouldRun(cpath string) (ok bool, err error) {
	// look if ECL rules are activated
	if !ecl.Activated {
		return true, nil
	}

	fp, err := os.Open(cpath)
	if err != nil {
		return false, err
	}

	return shouldRun(ecl, fp)
}

// ShouldRunFp determines if an already opened container should run according to its execgroup rules
func (ecl *EclConfig) ShouldRunFp(fp *os.File) (ok bool, err error) {
	// look if ECL rules are activated
	if !ecl.Activated {
		return true, nil
	}

	return shouldRun(ecl, fp)
}
