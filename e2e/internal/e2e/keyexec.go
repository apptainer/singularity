// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package e2e

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var importPrivKeyScript string

func getImportScript(kpath string) string {
	// Yes, this uses /usr/bin/expect
	return fmt.Sprintf(`
set timeout -1

set psk "e2etests"

spawn singularity key import %s

expect "Enter your old password : "
send "${psk}\r"

expect "Enter a new password for this key : "
send "${psk}\r"

expect "Retype your passphrase : "
send "${psk}\r"

expect eof
`, kpath)
}

func getExportScript(num int, kpath, armor, psk string) string {
	// Yes, this uses /usr/bin/expect
	return fmt.Sprintf(`
set timeout -1

spawn singularity key export --secret %s %s

expect "Enter # of signing key to use : "
send "%v\r"

expect "Enter key passphrase : "
send "%s\r"

expect eof
`, armor, kpath, num, psk)
}

var backupSypgp = filepath.Join(HomeDir(), ".singularity/sypgp/secret-keyring-backup")

// PullDefaultPublicKey will pull the public Sylabs Admin key
func PullDefaultPublicKey(t *testing.T) {
	LoadEnv(t, &testenv)

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
	LoadEnv(t, &testenv)

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
	err := os.Rename(filepath.Join(HomeDir(), ".singularity/sypgp/pgp-secret"), backupSypgp)
	if err != nil {
		t.Fatalf("Unable to rename secret keyring: %v", err)
	}
}

// RecoverSecretKeyring will recover your secret keyring, this gets ran after the private key test are complete.
func RecoverSecretKeyring(t *testing.T) {
	if err := os.Remove(filepath.Join(HomeDir(), ".singularity/sypgp/pgp-secret")); err != nil {
		t.Fatalf("Unable to remove secret keyring: %v", err)
	}
	if err := os.Rename(backupSypgp, filepath.Join(HomeDir(), ".singularity/sypgp/pgp-secret")); err != nil {
		t.Fatalf("Unable to rename secret keyring: %v", err)
	}
}

// RemoveSecretKeyring will delete your secret keyring :O
func RemoveSecretKeyring(t *testing.T) {
	err := os.Remove(filepath.Join(HomeDir(), ".singularity/sypgp/pgp-secret"))
	if err != nil {
		t.Fatalf("Unable to remove secret keyring: %v", err)
	}
}

// ImportKey will import a key from kpath.
func ImportKey(t *testing.T, kpath string) ([]byte, error) {
	LoadEnv(t, &testenv)

	argv := []string{"key", "import", kpath}
	execKey := exec.Command(testenv.CmdPath, argv...)

	return execKey.CombinedOutput()
}

// ImportTestKey will take a private key file (kpath) and import it.
func ImportPrivateKey(t *testing.T, kpath string) ([]byte, error) {
	s := getImportScript(kpath)

	importScript, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Unable to create script: %v", err)
	}
	defer importScript.Close()

	err = ioutil.WriteFile(importScript.Name(), []byte(s), 0644)
	if err != nil {
		t.Fatalf("Unable to write tmp file: %v", err)
	}

	argv := []string{importScript.Name()}
	execImport := exec.Command("/usr/bin/expect", argv...)

	return execImport.CombinedOutput()
}

// ExportPrivateKey will import a private key from kpath.
func ExportPrivateKey(t *testing.T, kpath string) ([]byte, error) {
	s := getExportScript(0, kpath, "", "e2etests")

	exportScript, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Unable to create script: %v", err)
	}
	defer exportScript.Close()

	_, err = exportScript.WriteString(s)
	if err != nil {
		t.Fatalf("Unable to write to file: %v", err)
	}
	err = exportScript.Close()
	if err != nil {
		t.Fatalf("Unable to clise file: %v", err)
	}

	argv := []string{exportScript.Name()}
	execImport := exec.Command("/usr/bin/expect", argv...)

	return execImport.CombinedOutput()
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
