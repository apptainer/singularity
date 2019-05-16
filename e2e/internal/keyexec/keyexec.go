// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keyexec

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/pkg/sypgp"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

// PullDefaultPublicKey will pull the public Sylabs Admin key
func PullDefaultPublicKey(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key", "pull", "F69C21F759C8EA06FD32CCF4536523CE1E109AF3"}

	execKey := exec.Command(testenv.CmdPath, argv...)

	out, err := execKey.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("Unable to pull key: %v", err)
	}
}

// RemoveDefaultPublicKey will pull the public Sylabs Admin key
func RemoveDefaultPublicKey(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key", "remove", "F69C21F759C8EA06FD32CCF4536523CE1E109AF3"}
	execKey := exec.Command(testenv.CmdPath, argv...)

	out, err := execKey.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("Unable to pull key: %v", err)
	}
}

// BackupSecretKeyring will take your secret keyring file, and back it up. This gets ran before the private
// key testing.
func BackupSecretKeyring(t *testing.T) {
	backupSypgp := filepath.Join(sypgp.DirPath(), "secret-keyring-backup")

	err := os.Rename(sypgp.SecretPath(), backupSypgp)
	if err != nil {
		t.Fatalf("Unable to rename secret keyring: %v", err)
	}
}

// RecoverSecretKeyring will recover your secret keyring, this gets ran after the private key test are complete.
func RecoverSecretKeyring(t *testing.T) {
	backupSypgp := filepath.Join(sypgp.DirPath(), "secret-keyring-backup")

	if err := os.Remove(sypgp.SecretPath()); err != nil {
		t.Fatalf("Unable to remove secret keyring: %v", err)
	}
	if err := os.Rename(backupSypgp, sypgp.SecretPath()); err != nil {
		t.Fatalf("Unable to rename secret keyring: %v", err)
	}
}

// RemoveSecretKeyring will delete your secret keyring.
func RemoveSecretKeyring(t *testing.T) {
	err := os.Remove(sypgp.SecretPath())
	if err != nil {
		t.Fatalf("Unable to remove secret keyring: %v", err)
	}
}

// ImportKey will import a key from kpath.
func ImportKey(t *testing.T, kpath string) ([]byte, error) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key", "import", kpath}
	execKey := exec.Command(testenv.CmdPath, argv...)

	return execKey.CombinedOutput()
}

// ImportPrivateKey will take a private key file (kpath) and import it.
func ImportPrivateKey(t *testing.T, kpath string) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole(expect.WithStdin(os.Stdin))
	if err != nil {
		panic(err)
	}
	defer c.Close()

	exportCmd := []string{"key", "import", kpath}

	cmd := exec.Command(testenv.CmdPath, exportCmd...)
	cmd.Stdin = c.Tty()

	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	cmd.Stdout = buf

	go func() {
		c.ExpectEOF()
	}()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	// Send the passcode to singularity
	c.Send("e2etests\n")
	c.Send("e2etests\n")
	c.Send("e2etests\n")

	err = cmd.Wait()
	cm := fmt.Sprintf("%s %s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, buf.Bytes(), err
}

// ExportPrivateKey will import a private key from kpath.
func ExportPrivateKey(t *testing.T, kpath, num string, armor bool) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole(expect.WithStdin(os.Stdin))
	if err != nil {
		panic(err)
	}
	defer c.Close()

	exportCmd := []string{"key", "export", "--secret"}

	if armor {
		exportCmd = append(exportCmd, "--armor")
	}

	exportCmd = append(exportCmd, kpath)

	outErr := bytes.NewBuffer(nil)

	cmd := exec.Command(testenv.CmdPath, exportCmd...)
	cmd.Stdin = c.Tty()

	cmd.Stderr = outErr
	cmd.Stdout = outErr

	go func() {
		c.ExpectEOF()
	}()

	err = cmd.Start()
	if err != nil {
		t.Fatalf("unable to run command: %v", err)
	}

	c.Send(num)
	c.Send("e2etests\n")

	err = cmd.Wait()
	cm := fmt.Sprintf("%s %s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, outErr.Bytes(), err
}

// CorruptKey will take a ASCII key (kpath) and change some chars in it (corrupt it).
func CorruptKey(t *testing.T, kpath string) {
	input, err := ioutil.ReadFile(kpath)
	if err != nil {
		t.Fatalf("Unable to read file: %v", err)
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		if strings.Contains(line, "B") {
			lines[i] = "P"
		}
	}
	output := strings.Join(lines, "\n")
	err = ioutil.WriteFile(kpath, []byte(output), 0644)
	if err != nil {
		t.Fatalf("Unable to write to file: %v", err)
	}
}

// RunKeyCmd will run a 'singularty key' command, with any args that are set in commands.
func RunKeyCmd(t *testing.T, cmdPath string, commands []string, file, stdin string) (string, []byte, error) {
	argv := []string{"key"}
	argv = append(argv, commands...)

	if file != "" {
		argv = append(argv, file)
	}

	cmd := fmt.Sprintf("%s %s", cmdPath, strings.Join(argv, " "))
	execKey := exec.Command(cmdPath, argv...)

	stdinRun, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer stdinRun.Close()

	_, err = io.WriteString(stdinRun, stdin)
	if err != nil {
		t.Fatalf("%v", err)
	}

	_, err = stdinRun.Seek(0, os.SEEK_SET)
	if err != nil {
		t.Fatalf("%v", err)
	}

	execKey.Stdin = stdinRun
	out, err := execKey.CombinedOutput()

	return cmd, out, err
}
