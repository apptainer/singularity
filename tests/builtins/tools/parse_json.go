package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/sylabs/singularity/pkg/stest"
	"mvdan.cc/sh/v3/interp"
)

// parse-json builtin
// usage:
// parse-json "{{.a}}" /tmp/file.json
// echo '{"a": "value"}' | parse-json "{{.a}}"
func parseJSON(ctx context.Context, mc interp.ModuleCtx, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("parse-json requires at least one argument")
	}
	tpl := args[0]
	t := template.Must(template.New("").Parse(tpl))
	if t == nil {
		return fmt.Errorf("failed to parse template %q", tpl)
	}

	var err error
	var data []byte

	if len(args) == 2 {
		data, err = ioutil.ReadFile(args[1])
		if err != nil {
			return fmt.Errorf("failed to open file %s: %s", args[1], err)
		}
	} else {
		data, err = ioutil.ReadAll(mc.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read JSON from stdin: %s", err)
		}
	}

	m := map[string]interface{}{}
	if err = json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("failed to unmarshal JSON data: %s", err)
	}
	if err = t.Execute(mc.Stdout, m); err != nil {
		return fmt.Errorf("failed to apply template: %s", err)
	}

	return nil
}

func init() {
	stest.RegisterCommandBuiltin("parse-json", parseJSON)
}
