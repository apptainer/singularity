// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package keyexec

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	expect "github.com/Netflix/go-expect"
	"github.com/sylabs/singularity/e2e/internal/e2e"
	"github.com/sylabs/singularity/internal/pkg/test"
	"github.com/sylabs/singularity/pkg/sypgp"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

// E2ePrivatePass is the password used for importing/exportin private keys.
// Make sure theres a newline after the password.
const E2ePrivatePass = "e2etests"

// E2eKeyFingerprint is the e2e test key fingerprint.
const E2eKeyFingerprint = "F69C21F759C8EA06FD32CCF4536523CE1E109AF3"

// PullDefaultPublicKey will pull the public E2E test key.
func PullDefaultPublicKey(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key", "pull", E2eKeyFingerprint}

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

	argv := []string{"key", "remove", E2eKeyFingerprint}
	execKey := exec.Command(testenv.CmdPath, argv...)

	out, err := execKey.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Fatalf("Unable to pull key: %v", err)
	}
}

func removePublicKeyring(t *testing.T) {
	err := os.Remove(sypgp.PublicPath())
	if err != nil {
		t.Fatalf("Unable to remove public keyring: %v", err)
	}
}

// RemoveSecretKeyring will delete your secret keyring.
func RemoveSecretKeyring(t *testing.T) {
	err := os.Remove(sypgp.SecretPath())
	if err != nil {
		t.Fatalf("Unable to remove secret keyring: %v", err)
	}
}

func RemoveKeyring(t *testing.T) {
	err := os.RemoveAll(sypgp.DirPath())
	if err != nil {
		t.Fatalf("Unable to remove keyring directory: %v", err)
	}
}

// ImportKey will import a key from kpath.
func ImportKey(t *testing.T, kpath string) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key", "import", kpath}
	execKey := exec.Command(testenv.CmdPath, argv...)

	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(argv, " "))

	b, err := execKey.CombinedOutput()

	return cm, b, err
}

// ImportPrivateKey will take a private key file (kpath) and import it.
func ImportPrivateKey(t *testing.T, kpath string) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatal("Unable to start new console: ", err)
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
		t.Fatal("Unable to start command: ", err)
	}

	// Send the passcode to singularity. The first one is the old
	// password, the next two are the new passowrd.
	c.Send(E2ePrivatePass + "\n")
	c.Send(E2ePrivatePass + "\n")
	c.Send(E2ePrivatePass + "\n")

	err = cmd.Wait()
	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, buf.Bytes(), err
}

// ExportPrivateKey will import a private key from kpath.
func ExportPrivateKey(t *testing.T, kpath, num string, armor bool) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatal("Unable to start new console: ", err)
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
	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, outErr.Bytes(), err
}

// RunKeyCmd will run a 'singularty key' command, with any args that are set in commands.
func RunKeyCmd(t *testing.T, cmdPath string, commands []string, stdin string) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key"}
	argv = append(argv, commands...)

	cm := fmt.Sprintf("%s\n%s", cmdPath, strings.Join(argv, " "))
	execKey := exec.Command(cmdPath, argv...)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatal("Unable to start new console: ", err)
	}
	defer c.Close()

	outErr := bytes.NewBuffer(nil)

	execKey.Stdin = c.Tty()

	execKey.Stderr = outErr
	execKey.Stdout = outErr

	go func() {
		c.ExpectEOF()
	}()

	err = execKey.Start()
	if err != nil {
		t.Fatalf("unable to run command: %v", err)
	}

	c.Send(stdin)

	err = execKey.Wait()

	return cm, outErr.Bytes(), err
}

// QuickTestExportImportKey will export a private, and public key (0), and then import them. This is used after
// generating a newpair.
func QuickTestExportImportKey(t *testing.T) {
	e2e.LoadEnv(t, &testenv)

	tmpTestDir := filepath.Join(testenv.TestDir, "quick_test_key_verify")

	tests := []struct {
		name    string
		private bool
		armor   bool
		succeed bool
	}{
		{
			name:    "quick test public",
			private: false,
			armor:   false,
			succeed: true,
		},
		{
			name:    "quick test public armor",
			private: false,
			armor:   true,
			succeed: true,
		},
		{
			name:    "quick test private",
			private: true,
			armor:   false,
			succeed: true,
		},
		{
			name:    "quick test private armor",
			private: true,
			armor:   true,
			succeed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			os.RemoveAll(tmpTestDir)
			os.MkdirAll(tmpTestDir, os.ModePerm)
			var c string
			var b []byte
			var err error

			if tt.private {
				c, b, err = ExportPrivateKey(t, filepath.Join(tmpTestDir, "export_key.asc"), "0\n", tt.armor)
			} else {
				c, b, err = RunKeyCmd(t, testenv.CmdPath, []string{"export", filepath.Join(tmpTestDir, "export_key.asc")}, "0\n")
			}
			if tt.succeed {
				if err != nil {
					t.Log("Command that failed: ", c)
					t.Log(string(b))
					t.Fatalf("unexpected failure: %v", err)
				}
				if tt.private {
					t.Run("remove_private_keyring_before_importing", test.WithoutPrivilege(func(t *testing.T) { RemoveSecretKeyring(t) }))
				} else {
					t.Run("remove_public_keyring_before_importing", test.WithoutPrivilege(func(t *testing.T) { removePublicKeyring(t) }))
				}
				t.Run("import_private_keyring_from", test.WithoutPrivilege(func(t *testing.T) {
					c, b, err := ImportPrivateKey(t, filepath.Join(tmpTestDir, "export_key.asc"))
					if err != nil {
						t.Log("command that failed: ", c, string(b))
						t.Fatalf("Unable to import key: %v", err)
					}
				}))
			} else {
				if err == nil {
					t.Log(string(b))
					t.Fatalf("unexpected success running: %v", c)
				}
			}
		}))
	}
}

// KeyNewPair will generate a newpair with the arguments being the key information; user = username, email = email, etc...
// Will return a command that ran (string), the output of the command, and the error.
func KeyNewPair(t *testing.T, user, email, note, psk1, psk2 string, push bool) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatalf("Unable to open new console: %v", err)
	}
	defer c.Close()

	exportCmd := []string{"key", "newpair"}
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

	c.Send(user)
	c.Send(email)
	c.Send(note)
	c.Send(psk1)
	if psk2 != "" {
		c.Send(psk2)
	} else {
		c.Send(psk1)
	}
	// TODO: Make sure CCI/Travis has an access token before pushing
	if push {
		c.Send("y\n")
	} else {
		c.Send("n\n")
	}

	err = cmd.Wait()
	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, outErr.Bytes(), err

}
