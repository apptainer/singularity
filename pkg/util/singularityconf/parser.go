// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package singularityconf

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// Directives represents the configuration directives type
// holding directives mapped to their respective values.
type Directives map[string][]string

var parserReg = regexp.MustCompile(`(?m)^\s*([a-zA-Z _-]+)[[:blank:]]*=[[:blank:]]*(.*)$`)

// GetDirectives parses configuration directives from reader
// and returns a directive map with associated values.
func GetDirectives(reader io.Reader) (Directives, error) {
	if reader == nil {
		return make(Directives), nil
	}

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("while reading data: %s", err)
	}

	directives := make(Directives)

	for _, match := range parserReg.FindAllSubmatch(data, -1) {
		if match != nil {
			key := strings.TrimSpace(string(match[1]))
			val := strings.TrimSpace(string(match[2]))
			if val != "" {
				directives[key] = append(directives[key], val)
			}
		}
	}

	return directives, nil
}

// HasDirective returns if the directive is present or not.
func HasDirective(directive string) bool {
	if directive == "" {
		return false
	}

	file := new(File)
	elem := reflect.ValueOf(file).Elem()

	for i := 0; i < elem.NumField(); i++ {
		typeField := elem.Type().Field(i)

		if typeField.Tag.Get("directive") == directive {
			return true
		}
	}

	return false
}

// GetConfig sets the corresponding interface fields associated
// with directives.
func GetConfig(directives Directives) (*File, error) {
	file := new(File)

	elem := reflect.ValueOf(file).Elem()

	// Iterate over the fields of f and handle each type
	for i := 0; i < elem.NumField(); i++ {
		valueField := elem.Field(i)
		typeField := elem.Type().Field(i)

		dir, ok := typeField.Tag.Lookup("directive")
		if !ok {
			return nil, fmt.Errorf("no directive tag found for field %q", typeField.Name)
		}

		defaultValue := ""
		if v, ok := typeField.Tag.Lookup("default"); ok {
			defaultValue = v
		}

		authorized := []string{}
		if v, ok := typeField.Tag.Lookup("authorized"); ok {
			authorized = strings.Split(v, ",")
		}

		kind := typeField.Type.Kind()

		value := []string{}
		if len(directives[dir]) > 0 {
			for _, dv := range directives[dir] {
				if dv != "" {
					value = append(value, strings.Split(dv, ",")...)
				}
			}
		} else {
			if defaultValue != "" && (kind != reflect.Slice || directives == nil) {
				value = append(value, strings.Split(defaultValue, ",")...)
			}
		}

		switch kind {
		case reflect.Bool:
			found := false
			for _, a := range authorized {
				if a == value[0] {
					found = true
					break
				}
			}
			if !found && len(authorized) > 0 {
				return nil, fmt.Errorf("value authorized for directive %q are %s", dir, authorized)
			}
			valueField.SetBool(value[0] == "yes")
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(value[0], 0, 64)
			if err != nil {
				return nil, err
			}
			valueField.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			n, err := strconv.ParseUint(value[0], 0, 64)
			if err != nil {
				return nil, err
			}
			valueField.SetUint(n)
		case reflect.String:
			if len(value) == 0 {
				value = []string{""}
			}
			found := false
			for _, a := range authorized {
				if a == value[0] {
					found = true
					break
				}
			}
			if !found && len(authorized) > 0 && value[0] != "" {
				return nil, fmt.Errorf("value authorized for directive '%s' are %s", dir, authorized)
			}
			valueField.SetString(value[0])
		case reflect.Slice:
			l := len(value)
			v := reflect.MakeSlice(typeField.Type, l, l)
			valueField.Set(v)

			switch t := valueField.Interface().(type) {
			case []string:
				for i, val := range value {
					t[i] = strings.TrimSpace(val)
				}
			}
		}
	}

	return file, nil
}

// Parse parses configuration file with the specified path.
func Parse(filepath string) (*File, error) {
	if filepath == "" {
		// grab the default configuration
		return GetConfig(nil)
	}

	c, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer c.Close()

	directives, err := GetDirectives(c)
	if err != nil {
		return nil, fmt.Errorf("while parsing data: %s", err)
	}

	return GetConfig(directives)
}

// Generate executes the default template asset on File object if
// no custom template path is provided otherwise it uses the template
// found in the path.
func Generate(out io.Writer, tmplPath string, config *File) error {
	var err error
	var t *template.Template

	if tmplPath != "" {
		t, err = template.ParseFiles(tmplPath)
		if err != nil {
			return fmt.Errorf("unable to parse template %s: %s", tmplPath, err)
		}
	} else {
		t, err = template.New("singularity.conf").Parse(TemplateAsset)
		if err != nil {
			return fmt.Errorf("unable to create template: %s", err)
		}
	}

	if err := t.Execute(out, config); err != nil {
		return fmt.Errorf("unable to execute template text for %s on %v: %v", t.Name(), config, err)
	}

	return nil
}
