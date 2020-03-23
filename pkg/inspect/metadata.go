// Copyright (c) 2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package inspect

// ContainerType defines the container type (used by default).
const ContainerType = "container"

// AppAttributes describes app metadata attributes.
type AppAttributes struct {
	Environment map[string]string `json:"environment,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Runscript   string            `json:"runscript,omitempty"`
	Test        string            `json:"test,omitempty"`
	Helpfile    string            `json:"helpfile,omitempty"`
}

// Attributes describes metadata attributes of Singularity containers.
type Attributes struct {
	Apps        map[string]*AppAttributes `json:"apps,omitempty"`
	Environment map[string]string         `json:"environment,omitempty"`
	Labels      map[string]string         `json:"labels,omitempty"`
	Runscript   string                    `json:"runscript,omitempty"`
	Test        string                    `json:"test,omitempty"`
	Helpfile    string                    `json:"helpfile,omitempty"`
	Deffile     string                    `json:"deffile,omitempty"`
	Startscript string                    `json:"startscript,omitempty"`
}

// Data holds the container metadata attributes.
type Data struct {
	Attributes Attributes `json:"attributes"`
}

// Metadata describes the JSON format of Singularity container metadata.
type Metadata struct {
	Data `json:"data"`
	Type string `json:"type"`
}

func (m *Metadata) AddApp(name string) {
	if _, ok := m.Attributes.Apps[name]; !ok {
		attr := &AppAttributes{}
		attr.Environment = make(map[string]string)
		attr.Labels = make(map[string]string)
		m.Attributes.Apps[name] = attr
	}
}

// NewMetadata returns an initialized instances of Metadata.
func NewMetadata() *Metadata {
	format := new(Metadata)
	format.Type = ContainerType
	format.Attributes.Labels = make(map[string]string)
	format.Attributes.Environment = make(map[string]string)
	format.Attributes.Apps = make(map[string]*AppAttributes)
	return format
}
