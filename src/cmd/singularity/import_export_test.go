// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package main

import (
	"compress/gzip"
	"io"
	"os"
	"os/exec"
)

type tarProcessor func(w io.WriteCloser) (io.WriteCloser, error)

// imageExport exports the image at imagePath, processes the TAR file using tarProcessor, and writes the result TAR file to exportPath.
func imageExport(imagePath, exportPath string, tp tarProcessor) error {

	// Open export file
	f, err := os.OpenFile(exportPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	// Wrap export file handle with TAR processor
	w, err := tp(f)
	if err != nil {
		return err
	}
	defer w.Close()

	// Run export command, copy output
	exportCmd := exec.Command(cmdPath, "image.export", imagePath)
	stdout, err := exportCmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := exportCmd.Start(); err != nil {
		return err
	}
	if _, err := io.Copy(w, stdout); err != nil {
		return err
	}
	if err := exportCmd.Wait(); err != nil {
		return err
	}
	return nil
}

// imageExportTAR exports the image at imagePath, and writes the result TAR
// file to exportPath.
func imageExportTAR(imagePath, exportPath string) error {
	return imageExport(imagePath, exportPath, func(w io.WriteCloser) (io.WriteCloser, error) {
		return w, nil
	})
}

// imageExportTGZ exports the image at imagePath, compresses it using gzip with
// the specified level of compression, and writes the result to exportPath.
func imageExportTGZ(imagePath, exportPath string, level int) error {
	return imageExport(imagePath, exportPath, func(w io.WriteCloser) (io.WriteCloser, error) {
		return gzip.NewWriterLevel(w, level)
	})
}
