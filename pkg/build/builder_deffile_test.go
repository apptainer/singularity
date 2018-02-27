/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.

  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

package build

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestCleanUpFile(t *testing.T) {
	testFiles := map[string]string{
		"docker":      "./mock/docker/docker",
		"debootstrap": "./mock/debootstrap/debootstrap",
		"arch":        "./mock/arch/arch",
		"yum":         "./mock/yum/yum",
		"shub":        "./mock/shub/shub",
		"localimage":  "./mock/localimage/localimage",
		"busybox":     "./mock/busybox/busybox",
		"zypper":      "./mock/zypper/zypper",
	}
	testResultFile := map[string]string{
		"docker":      "/tmp/singularity_test_build_docker",
		"debootstrap": "/tmp/singularity_test_build_debootstrap",
		"arch":        "/tmp/singularity_test_build_arch",
		"yum":         "/tmp/singularity_test_build_yum",
		"shub":        "/tmp/singularity_test_build_shub",
		"localimage":  "/tmp/singularity_test_build_localimage",
		"busybox":     "/tmp/singularity_test_build_busybox",
		"zypper":      "/tmp/singularity_test_build_zypper",
	}
	resultFile := map[string]string{
		"docker":      "./mock/docker/result",
		"debootstrap": "./mock/debootstrap/result",
		"arch":        "./mock/arch/result",
		"yum":         "./mock/yum/result",
		"shub":        "./mock/shub/result",
		"localimage":  "./mock/localimage/result",
		"busybox":     "./mock/busybox/result",
		"zypper":      "./mock/zypper/result",
	}
	type printSection func(*testing.T, *Deffile, *os.File)
	sectionsPrinters := map[string]printSection{
		"%help":        PrintHelpSection,
		"%setup":       PrintSetupSection,
		"%files":       PrintFilesSection,
		"%labels":      PrintLabelsSection,
		"%environment": PrintEnvSection,
		"%post":        PrintPostSection,
		"%runscript":   PrintRunscriptSection,
		"%test":        PrintTestSection,
	}

	// Loop through the Deffiles
	for k := range testFiles {
		t.Logf("=>\tRunning test for Deffile:\t\t[%s]", k)
		f, err := os.Create(testResultFile[k])
		if err != nil {
			t.Log(err)
			t.Fail()
		}

		defer f.Close()

		Df, err := DeffileFromPath(testFiles[k])
		if err != nil {
			t.Log(err)
			t.Fail()
		}
		//  `Write DeffileFromPath output to file
		for _, k := range headerKeys {
			v, ok := Df.Header[k]
			if ok {
				_, err := f.WriteString(fmt.Sprintf("%s:%s\n", k, v))
				if err != nil {
					t.Log(err)
					t.Fail()
				}
			}
		}
		for _, key := range sectionsKeys {
			printer := sectionsPrinters[key]
			printer(t, &Df, f)
		}
		// And....compare the output (fingers crossed)
		if !compareFiles(t, resultFile[k], testResultFile[k]) {
			t.Logf("<=\tFailed to parse Deffinition file:\t[%s]", k)
			t.Fail()
		}
	}
}

// compareFiles is a helper func to compare outputs
func compareFiles(t *testing.T, resultFile, testFile string) bool {
	rfile, err := os.Open(resultFile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer rfile.Close()

	tfile, err := os.Open(testFile)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer tfile.Close()

	rscanner := bufio.NewScanner(rfile)
	tscanner := bufio.NewScanner(tfile)
	for tscanner.Scan() {
		rscanner.Scan()
		rline := rscanner.Text()
		tline := tscanner.Text()
		if strings.Compare(rline, tline) != 0 {
			return false
		}
	}
	return true
}

func PrintHelpSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%help:%s\n", def.Sections.help)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintSetupSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%setup:%s\n", def.Sections.setup)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintPostSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%post:%s\n", def.Sections.post)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintRunscriptSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%runscript:%s\n", def.Sections.runscript)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintTestSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%test:%s\n", def.Sections.test)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintEnvSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := fmt.Sprintf("%%environment:%s\n", def.Sections.env)
	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintLabelsSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := "%labels" + "\n"

	for k, v := range def.Sections.labels {
		toPrint = toPrint + fmt.Sprintf("%s:%s\n", k, v)
	}

	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}

func PrintFilesSection(t *testing.T, def *Deffile, file *os.File) {
	toPrint := "%files" + "\n"

	for k, v := range def.Sections.files {
		toPrint = toPrint + fmt.Sprintf("%s:%s\n", k, v)
	}

	_, err := file.WriteString(toPrint)
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	// Issue a `Sync` to flush writes to stable storage.
	file.Sync()
}
