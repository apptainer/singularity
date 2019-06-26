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
		t.Fatalf("unable to pull key: %v", err)
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
		t.Fatalf("unable to pull key: %v", err)
	}
}

func removePublicKeyring(t *testing.T) {
	err := os.Remove(sypgp.PublicPath())
	if err != nil {
		t.Fatalf("unable to remove public keyring: %v", err)
	}
}

// RemoveSecretKeyring will delete your secret keyring.
func RemoveSecretKeyring(t *testing.T) {
	err := os.Remove(sypgp.SecretPath())
	if err != nil {
		t.Fatalf("unable to remove secret keyring: %v", err)
	}
}

func RemoveKeyring(t *testing.T) {
	err := os.RemoveAll(sypgp.DirPath())
	if err != nil {
		t.Fatalf("unable to remove keyring directory: %v", err)
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
		t.Fatal("unable to start new console: ", err)
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
		t.Fatal("unable to start command: ", err)
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
		t.Fatal("unable to start new console: ", err)
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
	c.Send(E2ePrivatePass + "\n")

	err = cmd.Wait()
	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(exportCmd, " "))

	return cm, outErr.Bytes(), err
}

// RunKeyCmd will run a 'singularty key' command, with any args that are set in commands.
func RunKeyCmd(t *testing.T, commands []string, stdin string) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	argv := []string{"key"}
	argv = append(argv, commands...)

	cm := fmt.Sprintf("%s\n%s", testenv.CmdPath, strings.Join(argv, " "))
	execKey := exec.Command(testenv.CmdPath, argv...)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatal("unable to start new console: ", err)
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
// generating a newpair. keyNum is the key number to test. Its a string, so the number must be used in "", and
// end with a '\n', eg. "1\n" will test key 1.
func QuickTestExportImportKey(t *testing.T, keyNum string) {
	e2e.LoadEnv(t, &testenv)

	tmpTestDir := filepath.Join(testenv.TestDir, "quick_test_key_verify")
	key := "export_key.asc"
	keyArmor := "export_key_armor.asc"
	keyPrivate := "export_key_private.asc"
	keyArmorPrivate := "export_key_armor_private.asc"

	tests := []struct {
		name    string
		private bool   // for private keys
		armor   bool   // for ASCII armor keys
		file    string // is the file that the key will be exported to
		succeed bool
	}{
		{
			name:    "quick test public",
			private: false,
			armor:   false,
			file:    key,
			succeed: true,
		},
		{
			name:    "quick test public armor",
			private: false,
			armor:   true,
			file:    keyArmor,
			succeed: true,
		},
		{
			name:    "quick test private",
			private: true,
			armor:   false,
			file:    keyPrivate,
			succeed: true,
		},
		{
			name:    "quick test private armor",
			private: true,
			armor:   true,
			file:    keyArmorPrivate,
			succeed: true,
		},
	}

	// Export the keys
	for _, tt := range tests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {
			os.MkdirAll(tmpTestDir, os.ModePerm)
			var c string
			var b []byte
			var err error

			if tt.private {
				// export the private key
				c, b, err = ExportPrivateKey(t, filepath.Join(tmpTestDir, tt.file), keyNum, tt.armor)
			} else {
				// export the public key
				cmd := []string{"export"}
				if tt.armor {
					cmd = append(cmd, "--armor")
				}
				cmd = append(cmd, filepath.Join(tmpTestDir, tt.file))
				c, b, err = RunKeyCmd(t, cmd, keyNum)
			}
			if tt.succeed {
				if err != nil {
					t.Log("command that failed: ", c, string(b))
					t.Fatalf("unexpected failure while exporting key: %v", err)
				}
			} else {
				if err == nil {
					t.Log(string(b))
					t.Fatalf("unexpected success running: %v", c)
				}
			}
		}))
	}

	// Import the keys
	importTests := []struct {
		name    string
		file    string
		private bool
	}{
		{
			name:    "import public binary",
			file:    key,
			private: false,
		},
		{
			name:    "import public ASCII",
			file:    keyArmor,
			private: false,
		},
		{
			name:    "import private binary",
			file:    keyPrivate,
			private: true,
		},
		{
			name:    "import private ASCII",
			file:    keyArmorPrivate,
			private: true,
		},
	}
	for _, tt := range importTests {
		t.Run(tt.name, test.WithoutPrivilege(func(t *testing.T) {

			if tt.private {
				RemoveSecretKeyring(t)

				c, b, err := ImportPrivateKey(t, filepath.Join(tmpTestDir, tt.file))
				if err != nil {
					t.Logf("command that failed: %s\n%s\n", c, string(b))
					t.Fatalf("unable to import key: %v", err)
				}
			} else {
				removePublicKeyring(t)

				c, b, err := ImportKey(t, filepath.Join(tmpTestDir, tt.file))
				if err != nil {
					t.Logf("command that failed: %s\n%s\n", c, string(b))
					t.Fatalf("unable to import key: %v", err)
				}
			}
		}))
	}
}

// KeyNewPair will generate a newpair with the arguments being the key information; user = username, email = email, etc...
// Will return a command that ran (string), the output of the command, and the error.
func KeyNewPair(t *testing.T, user, email, note, psk1 string, push bool) (string, []byte, error) {
	e2e.LoadEnv(t, &testenv)

	c, err := expect.NewConsole()
	if err != nil {
		t.Fatalf("unable to open new console: %v", err)
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
	c.Send(psk1)

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
