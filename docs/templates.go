// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package docs

// Global templates for help and usage strings
const (
	HelpTemplate string = `{{.Short}}

Usage:
  {{.UseLine}}

Description:{{.Long}}{{if .HasAvailableLocalFlags}}

Options:
{{.LocalFlags.FlagUsagesWrapped 80 | trimTrailingWhitespaces}}
{{end}}{{if .HasAvailableInheritedFlags}}

Global Options:
{{.InheritedFlags.FlagUsagesWrapped 80 | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}
Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasExample}}

Examples:{{.Example}}{{end}}


For additional help or support, please visit https://www.sylabs.io/docs/
`

	UseTemplate string = `Usage:
  {{TraverseParentsUses . | trimTrailingWhitespaces}}{{if .HasAvailableSubCommands}} <command>

Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}
`
)
