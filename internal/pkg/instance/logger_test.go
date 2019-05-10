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
	for _, f := range formatTest {
		logfile, err := ioutil.TempFile("", "log-")
		if err != nil {
			t.Errorf("failed to create temporary log file: %s", err)
		}
		fn := logfile.Name()
		// we only need the filename, not the file handle
		logfile.Close()

		logger, err := NewLogger(fn, f.formatter)
		if err != nil {
			t.Errorf("failed to create new logger: %s", err)
		}
		writer := logger.NewWriter(f.stream, f.dropCRNL)
		writer.Write([]byte(f.write))

		// close the writer end so that the reader gets an EOF
		// and both ends of the pipe are closed
		writer.Close()

		// call Sync so that we wait for the reader end to
		// finish and sync the write operations to the
		// destination file.
		logger.sync()

		d, err := ioutil.ReadFile(fn)
		if err != nil {
			t.Errorf("failed to read log data: %s", err)
		}

		if !bytes.Contains(d, []byte(f.search)) {
			t.Errorf("failed to retrieve %s in %s", f.search, string(d))
		}

		if err := os.Remove(fn); err != nil {
			t.Errorf("failed while deleting log file %s: %s", fn, err)
		}

		// will recreate log file
		logger.ReOpenFile()

		// delete it once again
		if err := os.Remove(fn); err != nil {
			t.Errorf("failed while deleting log file %s: %s", fn, err)
		}
	}
}
