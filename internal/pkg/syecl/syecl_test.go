// Copyright (c) 2020, Control Command Inc. All rights reserved.
// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package syecl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"gotest.tools/v3/golden"
)

const (
	KeyFP1 = "12045c8c0b1004d058de4beda20c27ee7ff7ba84"
	KeyFP2 = "7064B1D6EFF01B1262FED3F03581D99FE87EAFD1"
)

func TestAPutConfig(t *testing.T) {
	wl := Execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  "/var/data1",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := Execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  "/var/data2",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := Execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  "/var/data3",
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name string
		c    EclConfig
	}{
		{
			name: "Deactivated",
			c:    EclConfig{Activated: false},
		},
		{
			name: "DeactivatedLegacy",
			c:    EclConfig{Activated: false, Legacy: true},
		},
		{
			name: "Activated",
			c:    EclConfig{Activated: true},
		},
		{
			name: "ActivatedLegacy",
			c:    EclConfig{Activated: true, Legacy: true},
		},
		{
			name: "WhiteList",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name: "WhiteListLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name: "WhiteStrict",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name: "WhiteStrictLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name: "BlackList",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{bl}},
		},
		{
			name: "BlackListLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{bl}},
		},
		{
			name: "KitchenSink",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{wl, wls, bl}},
		},
		{
			name: "KitchenSinkLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wl, wls, bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tf, err := ioutil.TempFile("", "eclconfig-test")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tf.Name())
			tf.Close()

			if err := PutConfig(tt.c, tf.Name()); err != nil {
				t.Fatal(err)
			}

			b, err := ioutil.ReadFile(tf.Name())
			if err != nil {
				t.Fatal(err)
			}

			filename := filepath.Join(strings.Split(t.Name(), "/")...) + ".golden"
			golden.AssertBytes(t, b, filename)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	wl := Execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  "/var/data1",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := Execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  "/var/data2",
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := Execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  "/var/data3",
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name       string
		path       string
		wantConfig EclConfig
	}{
		{
			name:       "Deactivated",
			wantConfig: EclConfig{Activated: false},
		},
		{
			name:       "DeactivatedLegacy",
			wantConfig: EclConfig{Activated: false, Legacy: true},
		},
		{
			name:       "Activated",
			wantConfig: EclConfig{Activated: true},
		},
		{
			name:       "ActivatedLegacy",
			wantConfig: EclConfig{Activated: true, Legacy: true},
		},
		{
			name:       "WhiteList",
			wantConfig: EclConfig{Activated: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name:       "WhiteListLegacy",
			wantConfig: EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name:       "WhiteStrict",
			wantConfig: EclConfig{Activated: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name:       "WhiteStrictLegacy",
			wantConfig: EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name:       "BlackList",
			wantConfig: EclConfig{Activated: true, ExecGroups: []Execgroup{bl}},
		},
		{
			name:       "BlackListLegacy",
			wantConfig: EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{bl}},
		},
		{
			name:       "KitchenSink",
			wantConfig: EclConfig{Activated: true, ExecGroups: []Execgroup{wl, wls, bl}},
		},
		{
			name:       "KitchenSinkLegacy",
			wantConfig: EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wl, wls, bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join("testdata", "configs", fmt.Sprintf("%s.toml", tt.name))
			c, err := LoadConfig(path)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(c, tt.wantConfig) {
				t.Errorf("got config %v, want %v", c, tt.wantConfig)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	dirPath, err := filepath.Abs(filepath.Join("testdata", "images"))
	if err != nil {
		t.Fatal(err)
	}

	wl := Execgroup{
		TagName:  "name",
		ListMode: "whitelist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	wls := Execgroup{
		TagName:  "name",
		ListMode: "whitestrict",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1, KeyFP2},
	}
	bl := Execgroup{
		TagName:  "name",
		ListMode: "blacklist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1},
	}

	tests := []struct {
		name    string
		c       EclConfig
		wantErr bool
	}{
		{
			name: "DuplicatePaths",
			c: EclConfig{ExecGroups: []Execgroup{
				{DirPath: dirPath},
				{DirPath: dirPath},
			}},
			wantErr: true,
		},
		{
			name: "RelativePath",
			c: EclConfig{ExecGroups: []Execgroup{
				{DirPath: "testdata"},
			}},
			wantErr: true,
		},
		{
			name: "BadMode",
			c: EclConfig{ExecGroups: []Execgroup{
				{ListMode: "bad"},
			}},
			wantErr: true,
		},
		{
			name: "BadFingerprint",
			c: EclConfig{ExecGroups: []Execgroup{
				{ListMode: "whitelist", KeyFPs: []string{"bad"}},
			}},
			wantErr: true,
		},
		{
			name: "Deactivated",
			c:    EclConfig{Activated: false},
		},
		{
			name: "DeactivatedLegacy",
			c:    EclConfig{Activated: false, Legacy: true},
		},
		{
			name: "Activated",
			c:    EclConfig{Activated: true},
		},
		{
			name: "ActivatedLegacy",
			c:    EclConfig{Activated: true, Legacy: true},
		},
		{
			name: "WhiteList",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name: "WhiteListLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wl}},
		},
		{
			name: "WhiteStrict",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name: "WhiteStrictLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{wls}},
		},
		{
			name: "BlackList",
			c:    EclConfig{Activated: true, ExecGroups: []Execgroup{bl}},
		},
		{
			name: "BlackListLegacy",
			c:    EclConfig{Activated: true, Legacy: true, ExecGroups: []Execgroup{bl}},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.ValidateConfig(); (err != nil) != tt.wantErr {
				t.Errorf("got error %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// getTestEntity returns a fixed test PGP entity.
func getTestEntity(t *testing.T) *openpgp.Entity {
	t.Helper()

	f, err := os.Open(filepath.Join("testdata", "keys", "private.asc"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	el, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(el), 1; got != want {
		t.Fatalf("got %v entities, want %v", got, want)
	}
	return el[0]
}

func TestShouldRun(t *testing.T) {
	dirPath, err := filepath.Abs(filepath.Join("testdata", "images"))
	if err != nil {
		t.Fatal(err)
	}

	noDirPath1 := Execgroup{
		ListMode: "whitelist",
		DirPath:  "",
		KeyFPs:   []string{KeyFP1},
	}
	noDirPath2 := Execgroup{
		ListMode: "whitelist",
		DirPath:  "",
		KeyFPs:   []string{KeyFP2},
	}
	wl1 := Execgroup{
		ListMode: "whitelist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1},
	}
	wl2 := Execgroup{
		ListMode: "whitelist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP2},
	}
	ws1 := Execgroup{
		ListMode: "whitestrict",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1},
	}
	ws2 := Execgroup{
		ListMode: "whitestrict",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP2},
	}
	bl1 := Execgroup{
		ListMode: "blacklist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP1},
	}
	bl2 := Execgroup{
		ListMode: "blacklist",
		DirPath:  dirPath,
		KeyFPs:   []string{KeyFP2},
	}

	unsigned := filepath.Join(dirPath, "one-group.sif")
	signed := filepath.Join(dirPath, "one-group-signed.sif")
	legacySigned := filepath.Join(dirPath, "one-group-legacy-signed.sif")

	//nolint:maligned // the aligned form, with eg first, is not as easy to read
	tests := []struct {
		name      string
		activated bool
		legacy    bool
		eg        Execgroup
		path      string
		wantErr   bool
	}{
		{"BadListMode", true, false, Execgroup{ListMode: "bad"}, unsigned, true},
		{"Deactivated", false, false, Execgroup{}, unsigned, false},
		{"NoDirPathOK", true, false, noDirPath1, signed, false},
		{"NoDirPathError", true, false, noDirPath2, signed, true},
		{"WhitelistOK", true, false, wl1, signed, false},
		{"WhitelistError", true, false, wl2, signed, true},
		{"WhitestrictOK", true, false, ws1, signed, false},
		{"WhitestrictError", true, false, ws2, signed, true},
		{"BlacklistOK", true, false, bl2, signed, false},
		{"BlacklistError", true, false, bl1, signed, true},
		{"LegacyDeactivated", false, true, Execgroup{}, unsigned, false},
		{"LegacyNoDirPathOK", true, true, noDirPath1, legacySigned, false},
		{"LegacyNoDirPathError", true, true, noDirPath2, legacySigned, true},
		{"LegacyWhitelistOK", true, true, wl1, legacySigned, false},
		{"LegacyWhitelistError", true, true, wl2, legacySigned, true},
		{"LegacyWhitestrictOK", true, true, ws1, legacySigned, false},
		{"LegacyWhitestrictError", true, true, ws2, legacySigned, true},
		{"LegacyBlacklistOK", true, true, bl2, legacySigned, false},
		{"LegacyBlacklistError", true, true, bl1, legacySigned, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := EclConfig{
				Activated:  tt.activated,
				Legacy:     tt.legacy,
				ExecGroups: []Execgroup{tt.eg},
			}

			// Test ShouldRun (takes path).
			got, err := c.ShouldRun(tt.path, openpgp.EntityList{getTestEntity(t)})

			if want := !tt.wantErr; got != want {
				t.Errorf("got run %v, want %v", got, want)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("got err %v, wantErr %v", err, tt.wantErr)
			}

			// Test ShouldRun (takes file descriptor).
			f, err := os.Open(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			got, err = c.ShouldRunFp(f, openpgp.EntityList{getTestEntity(t)})

			if want := !tt.wantErr; got != want {
				t.Errorf("got run %v, want %v", got, want)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("got err %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
