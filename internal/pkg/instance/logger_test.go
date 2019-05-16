// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package instance

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

func TestLogger(t *testing.T) {
	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	formatTest := []struct {
		write     string
		stream    string
		dropCRNL  bool
		formatter LogFormatter
		search    string
	}{
		{
			write:     "test\n",
			stream:    "basic",
			formatter: LogFormats[BasicLogFormat],
			dropCRNL:  true,
			search:    " basic test",
		},
		{
			write:     "test\n",
			stream:    "k8s",
			formatter: LogFormats[KubernetesLogFormat],
			dropCRNL:  true,
			search:    " k8s F test",
		},
		{
			write:     "test\n",
			stream:    "json",
			formatter: LogFormats[JSONLogFormat],
			dropCRNL:  true,
			search:    "\"stream\":\"json\",\"log\":\"test\"",
		},
		{
			write:     "test\r\n",
			stream:    "basic",
			formatter: LogFormats[BasicLogFormat],
			dropCRNL:  false,
			search:    " basic test\\r\\n",
		},
		{
			write:    "test\n",
			stream:   "",
			dropCRNL: true,
			search:   " test",
		},
		{
			write:    "\n",
			stream:   "stdout",
			dropCRNL: true,
			search:   " stdout \n",
		},
		{
			write:    "",
			stream:   "stdout",
			dropCRNL: true,
			search:   "",
		},
	}

	logfile, err := ioutil.TempFile("", "log-")
	if err != nil {
		t.Errorf("failed to create temporary log file: %s", err)
	}
	filename := logfile.Name()
	logfile.Close()

	for _, f := range formatTest {
		logger, err := NewLogger(filename, f.formatter)
		if err != nil {
			t.Errorf("failed to create new logger: %s", err)
		}
		writer, err := logger.NewWriter(f.stream, f.dropCRNL)
		if err != nil {
			t.Errorf("failed to add new writer: %s", err)
		}
		writer.Write([]byte(f.write))

		logger.Close()

		d, err := ioutil.ReadFile(filename)
		if err != nil {
			t.Errorf("failed to read log data: %s", err)
		}

		if !bytes.Contains(d, []byte(f.search)) {
			t.Errorf("failed to retrieve %s in %s", f.search, string(d))
		}

		if err := os.Remove(filename); err != nil {
			t.Errorf("failed while deleting log file %s: %s", filename, err)
		}

		// will recreate log file
		if err := logger.ReOpenFile(); err != nil {
			t.Errorf("failed to reopen log file: %s", err)
		}
		// we call it again to just close re-opened log file descriptor
		logger.Close()

		// delete it once again
		if err := os.Remove(filename); err != nil {
			t.Errorf("failed while deleting log file %s: %s", filename, err)
		}
	}
}
