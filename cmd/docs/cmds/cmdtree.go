// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/cmd/internal/cli"
)

type Cmd struct {
	Name     string
	Options  []string
	Children []Cmd
}

func (c *Cmd) AddCmd(cmd Cmd) {
	c.Children = append(c.Children, cmd)
}

func (c *Cmd) AddOpt(opt string) {
	c.Options = append(c.Options, opt)
}

func buildTree(root *cobra.Command) Cmd {
	root.InitDefaultHelpFlag()

	tree := Cmd{Name: root.CommandPath(), Options: nil, Children: nil}

	root.Flags().VisitAll(func(flag *pflag.Flag) {
		tree.AddOpt(flag.Name)
	})

	for _, c := range root.Commands() {
		tree.AddCmd(buildTree(c))
	}

	return tree
}

func main() {
	cli.SingularityCmd.InitDefaultHelpCmd()
	cli.SingularityCmd.InitDefaultVersionFlag()

	tree := buildTree(cli.SingularityCmd)

	json, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(json))
}
