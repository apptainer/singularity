// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/cmd/internal/cli"
)

/*
 * This tool aims at gathering the list of Singularity commands and optionaly
 * the list of commands covered by the E2E tests. When gathering the list of
 * E2E tests, we also calculate the coverage compared to all the Singularity
 * commands.
 *
 * Note that the list of all the commands has some limitations. Since we create
 * the list with Cobra, the following assumptions are made:
 * - Commands are based on the following scheme (which is not matching reality at 100%):
 *		singularity <cmd> [[--<cmd-opt>] [<sub-cmd> [--<subcmd-opt]]]
 * - Cobra does not actually create a tree of commands/sub-commands/options but rather
 * a list of commands and sub-list of sub-commands and options. As a result, it is not
 * possible to track sub-command that are specific to a command option; and therefore
 * not possible to get all the actual Singularity commands. For instance, Singularity
 * support commands such as:
 *		singularity remote --config <configfile> add --global <name> <uri>
 * As the code stands, it is not possible to capture the fact that the config option
 * alloows one to use the add sub-command.
 * - the e2e tests are using the long version of the option, not the short version. At
 * the moment, we have no way to associate short and long versions of options.
 */

// Global variable to track the total number of tested commands.
var tested = 0

type singularityCmd struct {
	Name     string           `json:"name"`
	Options  []string         `json:"options"`
	Children []singularityCmd `json:"children"`
}

func checkCmd(sCmd string, e2eCmds string, resultFile *os.File) error {
	currentCmd := sCmd
	currentOpt := ""
	if strings.Index(sCmd, "--") > 0 {
		cmdElts := strings.Split(sCmd, "--")
		if len(cmdElts) != 2 {
			return fmt.Errorf("Wrong number of options: %d", len(cmdElts))
		}
		currentCmd = strings.Trim(cmdElts[0], " ")
		currentOpt = strings.Trim("--"+cmdElts[1], " ")
	}

	isTested := false
	for _, e2eCmd := range strings.Split(e2eCmds, "\n") {
		e2eCmd = strings.Trim(e2eCmd, " ")
		if e2eCmd == "" {
			continue
		}

		if e2eCmd != "" && strings.Contains(e2eCmd, currentCmd) {
			if currentOpt == "" || (currentOpt != "" && strings.Contains(e2eCmd, currentOpt)) {
				isTested = true
				break
			}
		}
	}

	str := fmt.Sprintf("UNTESTED: %s\n", sCmd)
	if isTested {
		fmt.Printf("%s%s%s\n", "\x1b[32m", sCmd, "\x1b[0m")
		str = fmt.Sprintf("TESTED: %s\n", sCmd)
		tested++
	} else {
		fmt.Printf("%s%s%s\n", "\x1b[31m", sCmd, "\x1b[0m")
	}
	resultFile.WriteString(str)

	return nil
}

func loadData(singularityCmdsFile, e2eCmdsFile string) (string, string, error) {
	e2eData, err := ioutil.ReadFile(e2eCmdsFile)
	if err != nil {
		return "", "", err
	}
	e2eCmds := string(e2eData)

	singularityData, err := ioutil.ReadFile(singularityCmdsFile)
	if err != nil {
		return "", "", err
	}
	singularityCmds := string(singularityData)

	return singularityCmds, e2eCmds, nil
}

func analyseData(singularityCmds string, e2eCmds string) (string, error) {
	resultFile, err := ioutil.TempFile("", "singularity-cmd-coverage-")
	if err != nil {
		return "", fmt.Errorf("failed to create file to store coverage results: %s", err)
	}
	defer resultFile.Close()

	for _, singularityCmd := range strings.Split(singularityCmds, "\n") {
		if singularityCmd != "" {
			err = checkCmd(singularityCmd, e2eCmds, resultFile)
			if err != nil {
				return "", fmt.Errorf("failed to check cmd %s", singularityCmd)
			}
		}
	}

	totalCmds := len(strings.Split(singularityCmds, "\n"))
	ratio := tested * 100 / totalCmds
	fmt.Printf("Coverage: %d/%d (%d%%)\n", tested, totalCmds, ratio)

	return resultFile.Name(), nil
}

func parseCmd(cmd singularityCmd, outputFile *os.File) error {
	str := fmt.Sprintf("%s", cmd)

	// When creating the tree of commands, it includes elements such as:
	//		{singularity cache [help] [{singularity cache clean [force help name type] []} {singularity cache list [help type verbose] []}]}
	// The fact that the element includes "[{" means that the first part is a command to
	// consider (here, "singularity cache [help]"), while the rest of the element is
	// a summary of commands already listed independently. This type of element is the
	// result of reaching to bottom of a tree branch.
	if strings.Contains(str, "[{") {
		str = strings.Split(str, "[{")[0]
	}
	cmds := strings.Split(str, "singularity")
	for _, cmdStr := range cmds { // This may be a command with sub-commands
		if cmdStr != "{" { // Going down the tree, we may be at the bottom with an empty list

			// We clean up the command
			cmdStr = strings.Replace(cmdStr, "[]}", "", -1)
			cmdStr = strings.Replace(cmdStr, "{", "", -1)
			cmdStr = strings.Replace(cmdStr, "]}", "", -1)
			cmdStr = strings.Replace(cmdStr, "] [", "]", -1)

			// We check whether the command has options to handle
			subcmds := strings.Split(cmdStr, "[")
			if len(subcmds) == 2 { // Options are associated to the command
				opts := subcmds[1]
				opts = strings.Replace(opts, "]", "", -1)
				// Handle each option separately
				for _, opt := range strings.Split(opts, " ") {
					if strings.Trim(opt, " \t") != "" {
						// Command with option
						str = fmt.Sprintf("singularity %s --%s\n", strings.Trim(subcmds[0], " \t"), strings.Trim(opt, " \t"))
						outputFile.WriteString(str)
					}
				}
			} else {
				// Command without option or sub-commands
				str = fmt.Sprintf("singularity %s\n", cmdStr)
				outputFile.WriteString(str)
			}
		}
	}

	return nil
}

func (c *singularityCmd) addCmd(cmd singularityCmd, outputFile *os.File) error {
	c.Children = append(c.Children, cmd)
	return parseCmd(cmd, outputFile)
}

func (c *singularityCmd) addOpt(opt string) {
	c.Options = append(c.Options, opt)
}

func buildTree(root *cobra.Command, outputFile *os.File) singularityCmd {
	root.InitDefaultHelpFlag()

	tree := singularityCmd{Name: root.CommandPath(), Options: nil, Children: nil}

	root.Flags().VisitAll(func(flag *pflag.Flag) {
		tree.addOpt(flag.Name)
	})

	for _, c := range root.Commands() {
		tree.addCmd(buildTree(c, outputFile), outputFile)
	}

	return tree
}

func createCmdsFiles() (string, string, error) {
	jsonFile, err := ioutil.TempFile("", "singularityCmdsJSON-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create command JSON file: %s", err)
	}
	defer jsonFile.Close()

	textFile, err := ioutil.TempFile("", "singularityCmds-")
	if err != nil {
		return "", "", fmt.Errorf("failed to create command text file: %s", err)
	}
	defer textFile.Close()
	cli.RootCmd().InitDefaultHelpCmd()
	cli.RootCmd().InitDefaultVersionFlag()

	tree := buildTree(cli.RootCmd(), textFile)

	json, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal data: %s", err)
	}

	jsonFile.WriteString(string(json))

	return jsonFile.Name(), textFile.Name(), nil
}

func runE2ETests() (string, error) {
	tempCoverageFileStr, err := ioutil.TempFile("", "e2e-cmds-")
	if err != nil {
		return "", fmt.Errorf("failed to create file to store e2e-cmds: %s", err)
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %s", err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command("make", "-C", "builddir", "e2e-test")
	coverageFileStr := "SINGULARITY_E2E_COVERAGE=" + tempCoverageFileStr.Name()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(os.Environ(), coverageFileStr)
	cmd.Dir = filepath.Join(dir, "../../..")
	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run E2E tests - stderr: %s; stdout: %s", stderr.String(), stdout.String())
	}

	return tempCoverageFileStr.Name(), nil
}

func main() {
	verbose := flag.Bool("v", false, "Enable verbose mode")
	coverage := flag.Bool("coverage", false, "Analyze the results")

	flag.Parse()

	jsonFilePath, textFilePath, err := createCmdsFiles()
	if err != nil {
		log.Fatalf("failed to gather all the Singularity commands: %s", err)
	}

	resultFilePath := ""
	e2eCmdsFilePath := ""
	if *coverage {
		e2eCmdsFilePath, err = runE2ETests()
		if err != nil {
			log.Fatalf("failed to run E2E tests: %s", err)
		}
		singularityCmds, e2eCmds, err := loadData(textFilePath, e2eCmdsFilePath)
		if err != nil {
			log.Fatalf("failed to load data: %s", err)
		}
		resultFilePath, err = analyseData(singularityCmds, e2eCmds)
		if err != nil {
			log.Fatalf("failed to analyze data: %s", err)
		}
	}

	if *verbose {
		fmt.Printf("List of all the Singularity commands (JSON) is in: %s\n", jsonFilePath)
		fmt.Printf("List of all the Singularity commands (text) is in: %s\n", textFilePath)
		if *coverage {
			fmt.Printf("List of all the Singularity commands currently tested by E2E: %s\n", e2eCmdsFilePath)
			fmt.Printf("Coverage results are saved in: %s\n", resultFilePath)
		}
	}
}
