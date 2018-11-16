// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package config

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

// Parser parses configuration found in the file with the specified path.
func Parser(filepath string, f interface{}) error {
	var c *os.File
	var b []byte
	directives := make(map[string][]string)

	if filepath != "" {
		c, err := os.Open(filepath)
		if err != nil {
			return err
		}
		b, err = ioutil.ReadAll(c)
		if err != nil {
			return err
		}

		c.Close()
	}

	r, err := regexp.Compile(`(?m)^\s*([a-zA-Z _]+)\s*=\s*(.*)$`)
	if err != nil {
		return fmt.Errorf("regex compilation failed")
	}

	for _, match := range r.FindAllSubmatch(b, -1) {
		if match != nil {
			key := strings.TrimSpace(string(match[1]))
			val := strings.TrimSpace(string(match[2]))
			directives[key] = append(directives[key], val)
		}
	}

	val := reflect.ValueOf(f).Elem()

	// Iterate over the fields of f and handle each type
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		def := typeField.Tag.Get("default")
		dir := typeField.Tag.Get("directive")
		authorized := strings.Split(typeField.Tag.Get("authorized"), ",")

		switch typeField.Type.Kind() {
		case reflect.Bool:
			found := false
			if directives[dir] != nil {
				for _, a := range authorized {
					if a == directives[dir][0] {
						if a == "yes" {
							valueField.SetBool(true)
						} else {
							valueField.SetBool(false)
						}
						found = true
						break
					}
				}
				if found == false {
					return fmt.Errorf("value authorized for directive '%s' are %s", dir, authorized)
				}
			} else {
				if def == "yes" {
					valueField.SetBool(true)
				} else {
					valueField.SetBool(false)
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var n int64
			var err error

			if directives[dir] != nil && directives[dir][0] != "" {
				n, err = strconv.ParseInt(directives[dir][0], 0, 64)
			} else {
				n, err = strconv.ParseInt(def, 0, 64)
			}
			if err != nil {
				return err
			}
			valueField.SetInt(n)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var n uint64
			var err error

			if directives[dir] != nil && directives[dir][0] != "" {
				n, err = strconv.ParseUint(directives[dir][0], 0, 64)
			} else {
				n, err = strconv.ParseUint(def, 0, 64)
			}
			if err != nil {
				return err
			}
			valueField.SetUint(n)
		case reflect.String:
			found := false
			if directives[dir] != nil {
				// To allow for string fields which are intended to be set to *any* value, we must
				// handle the case where authorized isn't set (implies any value is acceptable)
				if len(authorized) == 1 && authorized[0] == "" {
					valueField.SetString(directives[dir][0])
					found = true
				} else {
					for _, a := range authorized {
						if a == directives[dir][0] {
							valueField.SetString(a)
							found = true
							break
						}
					}
				}
				if found == false {
					return fmt.Errorf("value authorized for directive '%s' are %s", dir, authorized)
				}
			} else {
				valueField.SetString(def)
			}
		case reflect.Slice:
			l := len(directives[dir])
			switch valueField.Interface().(type) {
			case []string:
				if l == 1 {
					s := strings.Split(directives[dir][0], ",")
					l = len(s)
					if l != 1 {
						directives[dir] = s
					}
				} else if (l == 0 || c == nil) && def != "" {
					s := strings.Split(def, ",")
					l = len(s)
					directives[dir] = s
				}
				v := reflect.MakeSlice(typeField.Type, l, l)
				valueField.Set(v)
				t := valueField.Interface().([]string)
				for i, val := range directives[dir] {
					t[i] = strings.TrimSpace(val)
				}
			}
		}
	}
	return nil
}

// Generate executes the template stored at fpath on object f
func Generate(out io.Writer, tmplpath string, f interface{}) error {
	t, err := template.ParseFiles(tmplpath)
	if err != nil {
		return err
	}

	if err := t.Execute(out, f); err != nil {
		return fmt.Errorf("unable to execute template at %s on %v: %v", tmplpath, f, err)
	}

	return nil
}
