// Copyright (c) 2018-2021, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"context"
	"io"

	"github.com/hpcng/singularity/pkg/sylog"
	"github.com/vbauerster/mpb/v6"
	"github.com/vbauerster/mpb/v6/decor"
)

func initProgressBar(totalSize int64) (*mpb.Progress, *mpb.Bar) {
	p := mpb.New()

	if totalSize > 0 {
		return p, p.AddBar(totalSize,
			mpb.PrependDecorators(
				decor.Counters(decor.UnitKiB, "%.1f / %.1f"),
			),
			mpb.AppendDecorators(
				decor.Percentage(),
				decor.AverageSpeed(decor.UnitKiB, " % .1f "),
				decor.AverageETA(decor.ET_STYLE_GO),
			),
		)
	}
	return p, p.AddBar(totalSize,
		mpb.PrependDecorators(
			decor.Current(decor.UnitKiB, "%.1f / ???"),
		),
		mpb.AppendDecorators(
			decor.AverageSpeed(decor.UnitKiB, " % .1f "),
		),
	)
}

// See: https://ixday.github.io/post/golang-cancel-copy/
type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

// ProgressCallback is a function that provides progress information copying from a Reader to a Writer
type ProgressCallback func(int64, io.Reader, io.Writer) error

// ProgressBarCallback returns a progress bar callback unless e.g. --quiet or lower loglevel is set
func ProgressBarCallback(ctx context.Context) ProgressCallback {
	if sylog.GetLevel() <= -1 {
		// If we don't need a bar visible, we just copy data through the callback func
		return func(totalSize int64, r io.Reader, w io.Writer) error {
			return CopyWithContext(ctx, w, r)
		}
	}

	return func(totalSize int64, r io.Reader, w io.Writer) error {
		p, bar := initProgressBar(totalSize)

		// create proxy reader
		bodyProgress := bar.ProxyReader(r)
		defer bodyProgress.Close()

		err := CopyWithContext(ctx, w, bodyProgress)
		if err != nil {
			bar.Abort(true)
			return err
		}
		p.Wait()

		return nil
	}
}

func CopyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	// Copy will call the Reader and Writer interface multiple time, in order
	// to copy by chunk (avoiding loading the whole file in memory).
	// I insert the ability to cancel before read time as it is the earliest
	// possible in the call process.
	_, err := io.Copy(dst, readerFunc(func(p []byte) (int, error) {
		// golang non-blocking channel: https://gobyexample.com/non-blocking-channel-operations
		select {
		// if context has been canceled
		case <-ctx.Done():
			// stop process and propagate "context canceled" error
			return 0, ctx.Err()
		default:
			// otherwise just run default io.Reader implementation
			return src.Read(p)
		}
	}))
	return err
}

// DownloadProgressBar is used for chunked scs-library-client downloads.
type DownloadProgressBar struct {
	bar *mpb.Bar
	p   *mpb.Progress
}

func (pb *DownloadProgressBar) Init(contentLength int64) {
	if sylog.GetLevel() <= -1 {
		// we don't need a bar visible
		return
	}
	pb.p, pb.bar = initProgressBar(contentLength)
}

func (pb *DownloadProgressBar) ProxyReader(r io.Reader) io.ReadCloser {
	return pb.bar.ProxyReader(r)
}

func (pb *DownloadProgressBar) IncrBy(n int) {
	if pb.bar == nil {
		return
	}
	pb.bar.IncrBy(n)
}

func (pb *DownloadProgressBar) Abort(drop bool) {
	if pb.bar == nil {
		return
	}
	pb.bar.Abort(drop)
}

func (pb *DownloadProgressBar) Wait() {
	if pb.bar == nil {
		return
	}
	pb.p.Wait()
}
