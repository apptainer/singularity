// Copyright (c) 2018-2020, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"io"

	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/vbauerster/mpb/v4"
	"github.com/vbauerster/mpb/v4/decor"
)

// ProgressCallback is a function that provides progress information copying from a Reader to a Writer
type ProgressCallback func(int64, io.Reader, io.Writer) error

// ProgressBarCallback returns a progress bar callback unless e.g. --quiet or lower loglevel is set
func ProgressBarCallback() ProgressCallback {

	if sylog.GetLevel() <= -1 {
		return nil
	}

	return func(totalSize int64, r io.Reader, w io.Writer) error {
		p := mpb.New()
		bar := p.AddBar(totalSize,
			mpb.PrependDecorators(
				decor.Counters(decor.UnitKiB, "%.1f / %.1f"),
			),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.AverageSpeed(decor.UnitKiB, " % .1f "),
				decor.AverageETA(decor.ET_STYLE_GO),
			),
		)

		// create proxy reader
		bodyProgress := bar.ProxyReader(r)

		// Write the body to file
		_, err := io.Copy(w, bodyProgress)
		if err != nil {
			return err
		}

		return nil
	}
}
