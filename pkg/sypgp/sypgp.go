// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package sypgp implements the openpgp integration into the singularity project.
package sypgp

import (
	"bytes"
	"context"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/interactive"
	"github.com/sylabs/singularity/pkg/syfs"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	helpAuth = `Access token is expired or missing. To update or obtain a token:
  1) View configured remotes using "singularity remote list"
  2) Identify default remote. It will be listed with square brackets.
  3) Login to default remote with "singularity remote login <RemoteName>"
`
	helpPush = `  4) Push key using "singularity key push %[1]X"
`
)

var (
	errNotEncrypted = errors.New("key is not encrypted")

	// ErrEmptyKeyring is the error when the public, or private keyring
	// empty.
	ErrEmptyKeyring = errors.New("keyring is empty")
)

// KeyExistsError is a type representing an error associated to a specific key.
type KeyExistsError struct {
	fingerprint [20]byte
}

// Handle is a structure representing a keyring
type Handle struct {
	path string
}

// GenKeyPairOptions parameters needed for generating new key pair.
type GenKeyPairOptions struct {
	Name      string
	Email     string
	Comment   string
	Password  string
	KeyLength int
}

// mrKeyList contains all the key info, used for decoding
// the MR output from 'key search'
type mrKeyList struct {
	keyFingerprint string
	keyBit         string
	keyName        string
	keyType        string
	keyDateCreated string
	keyDateExpired string
	keyStatus      string
	keyCount       int
	keyReady       bool
}

func (e *KeyExistsError) Error() string {
	return fmt.Sprintf("the key with fingerprint %X already belongs to the keyring", e.fingerprint)
}

// GetTokenFile returns a string describing the path to the stored token file
func GetTokenFile() string {
	return filepath.Join(syfs.ConfigDir(), "sylabs-token")
}

// dirPath returns a string describing the path to the sypgp home folder
func dirPath() string {
	sypgpDir := os.Getenv("SINGULARITY_SYPGPDIR")
	if sypgpDir == "" {
		return filepath.Join(syfs.ConfigDir(), "sypgp")
	}
	return sypgpDir
}

// NewHandle initializes a new keyring in path.
func NewHandle(path string) *Handle {
	if path == "" {
		path = dirPath()
	}

	newHandle := new(Handle)
	newHandle.path = path

	return newHandle
}

// SecretPath returns a string describing the path to the private keys store
func (keyring *Handle) SecretPath() string {
	return filepath.Join(keyring.path, "pgp-secret")
}

// PublicPath returns a string describing the path to the public keys store
func (keyring *Handle) PublicPath() string {
	return filepath.Join(keyring.path, "pgp-public")
}

// ensureDirPrivate makes sure that the file system mode for the named
// directory does not allow other users access to it (neither read nor
// write).
//
// TODO(mem): move this function to a common location
func ensureDirPrivate(dn string) error {
	mode := os.FileMode(0700)

	oldumask := syscall.Umask(0077)

	err := os.MkdirAll(dn, mode)

	// restore umask...
	syscall.Umask(oldumask)

	// ... and check if there was an error in the os.MkdirAll call
	if err != nil {
		return err
	}

	dirinfo, err := os.Stat(dn)
	if err != nil {
		return err
	}

	if currentMode := dirinfo.Mode(); currentMode != os.ModeDir|mode {
		sylog.Warningf("Directory mode (%o) on %s needs to be %o, fixing that...", currentMode & ^os.ModeDir, dn, mode)
		if err := os.Chmod(dn, mode); err != nil {
			return err
		}
	}

	return nil
}

// createOrAppendPrivateFile creates the named filename, making sure
// it's only accessible to the current user.
//
// TODO(mem): move this function to a common location
func createOrAppendPrivateFile(fn string) (*os.File, error) {
	return os.OpenFile(fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
}

// ensureFilePrivate makes sure that the file system mode for the named
// file does not allow other users access to it (neither read nor
// write).
//
// TODO(mem): move this function to a common location
func ensureFilePrivate(fn string) error {
	mode := os.FileMode(0600)

	// just to be extra sure that we get the correct mode
	oldumask := syscall.Umask(0077)

	fs, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE, mode)

	// restore umask...
	syscall.Umask(oldumask)

	// ... and check if there was an error
	if err != nil {
		return err
	}
	defer fs.Close()

	// check and fix permissions
	fsinfo, err := fs.Stat()
	if err != nil {
		return err
	}

	if currentMode := fsinfo.Mode(); currentMode != mode {
		sylog.Warningf("File mode (%o) on %s needs to be %o, fixing that...", currentMode, fn, mode)
		if err := fs.Chmod(mode); err != nil {
			return err
		}
	}

	return nil
}

// PathsCheck creates the sypgp home folder, secret and public keyring files
func (keyring *Handle) PathsCheck() error {
	if err := ensureDirPrivate(keyring.path); err != nil {
		return err
	}

	if err := ensureFilePrivate(keyring.SecretPath()); err != nil {
		return err
	}

	if err := ensureFilePrivate(keyring.PublicPath()); err != nil {
		return err
	}

	return nil
}

func loadKeyring(fn string) (openpgp.EntityList, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return openpgp.ReadKeyRing(f)
}

// LoadPrivKeyring loads the private keys from local store into an EntityList
func (keyring *Handle) LoadPrivKeyring() (openpgp.EntityList, error) {
	if err := keyring.PathsCheck(); err != nil {
		return nil, err
	}

	return loadKeyring(keyring.SecretPath())
}

// LoadPubKeyring loads the public keys from local store into an EntityList
func (keyring *Handle) LoadPubKeyring() (openpgp.EntityList, error) {
	if err := keyring.PathsCheck(); err != nil {
		return nil, err
	}

	return loadKeyring(keyring.PublicPath())
}

// loadKeysFromFile loads one or more keys from the specified file.
//
// The key can be either a public or private key, and the file might be
// in binary or ascii armored format.
func loadKeysFromFile(fn string) (openpgp.EntityList, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	if entities, err := openpgp.ReadKeyRing(f); err == nil {
		return entities, nil
	}

	// cannot load keys from file, perhaps it's ascii armored?
	// rewind and try again
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return openpgp.ReadArmoredKeyRing(f)
}

// printEntity pretty prints an entity entry to w
func printEntity(w io.Writer, index int, e *openpgp.Entity) {
	// TODO(mem): this should not be here, this is presentation
	for _, v := range e.Identities {
		fmt.Fprintf(w, "%d) U: %s (%s) <%s>\n", index, v.UserId.Name, v.UserId.Comment, v.UserId.Email)
	}
	fmt.Fprintf(w, "   C: %s\n", e.PrimaryKey.CreationTime)
	fmt.Fprintf(w, "   F: %0X\n", e.PrimaryKey.Fingerprint)
	bits, _ := e.PrimaryKey.BitLength()
	fmt.Fprintf(w, "   L: %d\n", bits)

}

func printEntities(w io.Writer, entities openpgp.EntityList) {
	for i, e := range entities {
		printEntity(w, i, e)
		fmt.Fprint(w, "   --------\n")
	}
}

// PrintEntity pretty prints an entity entry
func PrintEntity(index int, e *openpgp.Entity) {
	printEntity(os.Stdout, index, e)
}

// PrintPubKeyring prints the public keyring read from the public local store
func (keyring *Handle) PrintPubKeyring() error {
	pubEntlist, err := keyring.LoadPubKeyring()
	if err != nil {
		return err
	}

	printEntities(os.Stdout, pubEntlist)

	return nil
}

// PrintPrivKeyring prints the secret keyring read from the public local store
func (keyring *Handle) PrintPrivKeyring() error {
	privEntlist, err := keyring.LoadPrivKeyring()
	if err != nil {
		return err
	}

	printEntities(os.Stdout, privEntlist)

	return nil
}

// storePrivKeys writes all the private keys in list to the writer w.
func storePrivKeys(w io.Writer, list openpgp.EntityList) error {
	for _, e := range list {
		if err := e.SerializePrivate(w, nil); err != nil {
			return err
		}
	}

	return nil
}

// appendPrivateKey appends a private key entity to the local keyring
func (keyring *Handle) appendPrivateKey(e *openpgp.Entity) error {
	f, err := createOrAppendPrivateFile(keyring.SecretPath())
	if err != nil {
		return err
	}
	defer f.Close()

	return storePrivKeys(f, openpgp.EntityList{e})
}

// storePubKeys writes all the public keys in list to the writer w.
func storePubKeys(w io.Writer, list openpgp.EntityList) error {
	for _, e := range list {
		if err := e.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

// appendPubKey appends a public key entity to the local keyring
func (keyring *Handle) appendPubKey(e *openpgp.Entity) error {
	f, err := createOrAppendPrivateFile(keyring.PublicPath())
	if err != nil {
		return err
	}
	defer f.Close()

	return storePubKeys(f, openpgp.EntityList{e})
}

// storePubKeyring overwrites the public keyring with the listed keys
func (keyring *Handle) storePubKeyring(keys openpgp.EntityList) error {
	f, err := os.OpenFile(keyring.PublicPath(), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, k := range keys {
		if err := k.Serialize(f); err != nil {
			return fmt.Errorf("could not store public key: %s", err)
		}
	}

	return nil
}

// compareKeyEntity compares a key ID with a string, returning true if the
// key and oldToken match.
func compareKeyEntity(e *openpgp.Entity, oldToken string) bool {
	// TODO: there must be a better way to do this...
	return fmt.Sprintf("%X", e.PrimaryKey.Fingerprint) == oldToken
}

func findKeyByFingerprint(entities openpgp.EntityList, fingerprint string) *openpgp.Entity {
	for _, e := range entities {
		if compareKeyEntity(e, fingerprint) {
			return e
		}
	}

	return nil
}

// CheckLocalPubKey will check if we have a local public key matching ckey string
// returns true if there's a match.
func (keyring *Handle) CheckLocalPubKey(ckey string) (bool, error) {
	// read all the local public keys
	elist, err := loadKeyring(keyring.PublicPath())
	switch {
	case os.IsNotExist(err):
		return false, nil

	case err != nil:
		return false, fmt.Errorf("unable to load local keyring: %v", err)
	}

	return findKeyByFingerprint(elist, ckey) != nil, nil
}

// removeKey removes one key identified by fingerprint from list.
//
// removeKey returns a new list with the key removed, or nil if the key
// was not found. The elements of the new list are the _same_ pointers
// found in the original list.
func removeKey(list openpgp.EntityList, fingerprint string) openpgp.EntityList {
	for idx, e := range list {
		if compareKeyEntity(e, fingerprint) {
			newList := make(openpgp.EntityList, len(list)-1)
			copy(newList, list[:idx])
			copy(newList[idx:], list[idx+1:])
			return newList
		}
	}

	return nil
}

// RemovePubKey will delete a public key matching toDelete
func (keyring *Handle) RemovePubKey(toDelete string) error {
	// read all the local public keys
	elist, err := loadKeyring(keyring.PublicPath())
	switch {
	case os.IsNotExist(err):
		return nil

	case err != nil:
		return fmt.Errorf("unable to list local keyring: %v", err)
	}

	newKeyList := removeKey(elist, toDelete)
	if newKeyList == nil {
		return fmt.Errorf("no key matching given fingerprint found")
	}

	sylog.Verbosef("Updating local keyring: %v", keyring.PublicPath())

	return keyring.storePubKeyring(newKeyList)
}

func (keyring *Handle) genKeyPair(opts GenKeyPairOptions) (*openpgp.Entity, error) {
	conf := &packet.Config{RSABits: opts.KeyLength, DefaultHash: crypto.SHA384}

	entity, err := openpgp.NewEntity(opts.Name, opts.Comment, opts.Email, conf)
	if err != nil {
		return nil, err
	}

	if opts.Password != "" {
		// Encrypt private key
		if err = EncryptKey(entity, opts.Password); err != nil {
			return nil, err
		}
	}

	// Store key parts in local key caches
	if err = keyring.appendPrivateKey(entity); err != nil {
		return nil, err
	}

	if err = keyring.appendPubKey(entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// GenKeyPair generates an PGP key pair and store them in the sypgp home folder
func (keyring *Handle) GenKeyPair(opts GenKeyPairOptions) (*openpgp.Entity, error) {
	if err := keyring.PathsCheck(); err != nil {
		return nil, err
	}

	entity, err := keyring.genKeyPair(opts)
	if err != nil {
		// Print the missing newline if there’s an error
		fmt.Printf("\n")
		return nil, err
	}

	return entity, nil
}

// DecryptKey decrypts a private key provided a pass phrase.
func DecryptKey(k *openpgp.Entity, message string) error {
	if message == "" {
		message = "Enter key passphrase : "
	}

	pass, err := interactive.AskQuestionNoEcho(message)
	if err != nil {
		return err
	}

	return k.PrivateKey.Decrypt([]byte(pass))
}

// EncryptKey encrypts a private key using a pass phrase
func EncryptKey(k *openpgp.Entity, pass string) error {
	if k.PrivateKey.Encrypted {
		return fmt.Errorf("key already encrypted")
	}
	return k.PrivateKey.Encrypt([]byte(pass))
}

// selectPubKey prints a public key list to user and returns the choice
func selectPubKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	if len(el) == 0 {
		return nil, ErrEmptyKeyring
	}
	printEntities(os.Stdout, el)

	n, err := interactive.AskNumberInRange(0, len(el)-1, "Enter # of public key to use : ")
	if err != nil {
		return nil, err
	}

	return el[n], nil
}

// SelectPrivKey prints a secret key list to user and returns the choice
func SelectPrivKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	if len(el) == 0 {
		return nil, ErrEmptyKeyring
	}
	printEntities(os.Stdout, el)

	n, err := interactive.AskNumberInRange(0, len(el)-1, "Enter # of private key to use : ")
	if err != nil {
		return nil, err
	}

	return el[n], nil
}

// formatMROutput will take a machine readable input, and convert it to fit
// on a 80x24 terminal. Returns the number of keys(int), the formated string
// in []bytes, and a error if one occurs.
func formatMROutput(mrString string) (int, []byte, error) {
	count := 0
	keyNum := 0
	listLine := "%s\t%s\t%s\n"

	retList := bytes.NewBuffer(nil)
	tw := tabwriter.NewWriter(retList, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, listLine, "KEY ID", "BITS", "NAME/EMAIL")

	key := strings.Split(mrString, "\n")

	for _, k := range key {
		nk := strings.Split(k, ":")
		for _, n := range nk {
			if n == "info" {
				var err error
				keyNum, err = strconv.Atoi(nk[2])
				if err != nil {
					return -1, nil, fmt.Errorf("unable to check key number")
				}
			}
			if n == "pub" {
				// The fingerprint is located at nk[1], and we only want the last 8 chars
				fmt.Fprintf(tw, "%s\t", nk[1][len(nk[1])-8:])
				// The key size (bits) is located at nk[3]
				fmt.Fprintf(tw, "%s\t", nk[3])
				count++
			}
			if n == "uid" {
				// And the key name/email is on nk[1]
				fmt.Fprintf(tw, "%s\t\n\n", nk[1])
			}
		}
	}
	tw.Flush()

	sylog.Debugf("key count=%d; expect=%d\n", count, keyNum)

	// Simple check to ensure the conversion was successful
	if count != keyNum {
		sylog.Debugf("expecting %d, got %d\n", keyNum, count)
		return -1, retList.Bytes(), fmt.Errorf("failed to convert machine readable to human readable output correctly")
	}

	return count, retList.Bytes(), nil
}

// SearchPubkey connects to a key server and searches for a specific key
func SearchPubkey(httpClient *http.Client, search, keyserverURI, authToken string, longOutput bool) error {
	// Get a Key Service client.
	c, err := client.NewClient(&client.Config{
		BaseURL:    keyserverURI,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		return err
	}

	// the max entities to print.
	pd := client.PageDetails{
		// still will only print 100 entities
		Size: 256,
	}

	// set the machine readable output on
	var options = []string{client.OptionMachineReadable}
	// Retrieve first page of search results from Key Service.
	keyText, err := c.PKSLookup(context.TODO(), &pd, search, client.OperationIndex, true, false, options)
	if err != nil {
		if jerr, ok := err.(*jsonresp.Error); ok && jerr.Code == http.StatusUnauthorized {
			// The request failed with HTTP code unauthorized. Guide user to fix that.
			sylog.Infof(helpAuth)
			return fmt.Errorf("unauthorized or missing token")
		} else if ok && jerr.Code == http.StatusNotFound {
			return fmt.Errorf("no matching keys found for fingerprint")
		} else {
			return fmt.Errorf("failed to get key: %v", err)
		}
	}

	if longOutput {
		kcount, keyList, err := formatMROutputLongList(keyText)
		fmt.Printf("Showing %d results\n\n%s", kcount, keyList)
		if err != nil {
			return fmt.Errorf("could not reformat key output")
		}
	} else {
		kcount, keyList, err := formatMROutput(keyText)
		fmt.Printf("Showing %d results\n\n%s", kcount, keyList)
		if err != nil {
			return err
		}
	}

	return nil
}

// getEncryptionAlgorithmName obtains the algorithm name for key encryption
func getEncryptionAlgorithmName(n string) (string, error) {
	algorithmName := ""

	code, err := strconv.ParseInt(n, 10, 64)
	if err != nil {
		return "", err
	}
	switch code {

	case 1, 2, 3:
		algorithmName = "RSA"
	case 16:
		algorithmName = "Elgamal"
	case 17:
		algorithmName = "DSA"
	case 18:
		algorithmName = "Elliptic Curve"
	case 19:
		algorithmName = "ECDSA"
	case 20:
		algorithmName = "Reserved"
	case 21:
		algorithmName = "Diffie-Hellman"
	default:
		algorithmName = "unknown"
	}
	return algorithmName, nil
}

//function to obtain a date format from linux epoch time
func date(s string) string {
	if s == "" {
		return "[ultimate]"
	}
	if s == "none" {
		return s
	}
	c, _ := strconv.ParseInt(s, 10, 64)
	ret := time.Unix(c, 0).String()

	return ret
}

// getKeyInfoFromList takes the lines, from strings.Split(), and a index of lines. Appends
// the output into keyList. Returns a error if one occurs.
func getKeyInfoFromList(keyList *mrKeyList, lines []string, index string) error {
	var errRet error

	if index == "pub" {
		// Get the fingerprint for the key
		keyList.keyFingerprint = lines[1]

		// Get the bit length for the key
		keyList.keyBit = lines[3]

		var err error
		// Get the key type
		keyList.keyType, err = getEncryptionAlgorithmName(lines[2])
		if err != nil {
			errRet = err
		}

		// Get the date created for the key
		keyList.keyDateCreated = date(lines[4])

		// Get the expiration date for the key
		keyList.keyDateExpired = date(lines[5])

		// Get the key status
		if lines[6] == "r" {
			keyList.keyStatus = "[revoked]"
		} else if lines[6] == "d" {
			keyList.keyStatus = "[disabled]"
		} else if lines[6] == "e" {
			keyList.keyStatus = "[expired]"
		} else {
			keyList.keyStatus = "[enabled]"
		}

		// Only count the key if it has a fingerprint. Otherwise
		// dont count it.
		keyList.keyCount++
	}
	if index == "uid" {
		// Get the name of the key
		keyList.keyName = lines[1]

		// After we get the name, the key is ready to print!
		keyList.keyReady = true
	}

	return errRet
}

// formatMROutputLongList reformats the key search output that is in machine readable format
// see the output format in: https://tools.ietf.org/html/draft-shaw-openpgp-hkp-00#section-5.2
func formatMROutputLongList(mrString string) (int, []byte, error) {
	listLine := "%s\t%s\t%s\t%s\t%s\t%s\t%s\n"

	retList := bytes.NewBuffer(nil)
	tw := tabwriter.NewWriter(retList, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, listLine, "FINGERPRINT", "ALGORITHM", "BITS", "CREATION DATE", "EXPIRATION DATE", "STATUS", "NAME/EMAIL")

	keyNum := 0
	key := strings.Split(mrString, "\n")
	var keyList mrKeyList

	for _, k := range key {
		nk := strings.Split(k, ":")
		for _, n := range nk {
			if n == "info" {
				var err error
				keyNum, err = strconv.Atoi(nk[2])
				if err != nil {
					return -1, nil, fmt.Errorf("unable to check key number")
				}
			}
			err := getKeyInfoFromList(&keyList, nk, n)
			if err != nil {
				return -1, nil, fmt.Errorf("failed to get entity from list: %s", err)
			}
		}
		if keyList.keyReady {
			fmt.Fprintf(tw, listLine, keyList.keyFingerprint, keyList.keyType, keyList.keyBit, keyList.keyDateCreated, keyList.keyDateExpired, keyList.keyStatus, keyList.keyName)
			fmt.Fprintf(tw, "\t\t\t\t\t\t\n")
			keyList = mrKeyList{keyCount: keyList.keyCount}
		}
	}
	tw.Flush()

	sylog.Debugf("key count=%d; expect=%d\n", keyList.keyCount, keyNum)

	// Simple check to ensure the conversion was successful
	if keyList.keyCount != keyNum {
		sylog.Debugf("expecting %d, got %d\n", keyNum, keyList.keyCount)
		return -1, retList.Bytes(), fmt.Errorf("failed to convert machine readable to human readable output correctly")
	}

	return keyList.keyCount, retList.Bytes(), nil
}

// FetchPubkey pulls a public key from the Key Service.
func FetchPubkey(httpClient *http.Client, fingerprint, keyserverURI, authToken string, noPrompt bool) (openpgp.EntityList, error) {

	// Decode fingerprint and ensure proper length.
	var fp []byte
	fp, err := hex.DecodeString(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to decode fingerprint: %v", err)
	}

	// theres probably a better way to do this
	if len(fp) != 4 && len(fp) != 20 {
		return nil, fmt.Errorf("not a valid key lenth: only accepts 8, or 40 chars")
	}

	// Get a Key Service client.
	c, err := client.NewClient(&client.Config{
		BaseURL:    keyserverURI,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		return nil, err
	}

	// Pull key from Key Service.
	keyText, err := c.GetKey(context.TODO(), fp)
	if err != nil {
		if jerr, ok := err.(*jsonresp.Error); ok && jerr.Code == http.StatusUnauthorized {
			// The request failed with HTTP code unauthorized. Guide user to fix that.
			sylog.Infof(helpAuth)
			return nil, fmt.Errorf("unauthorized or missing token")
		} else if ok && jerr.Code == http.StatusNotFound {
			return nil, fmt.Errorf("no matching keys found for fingerprint")
		} else {
			return nil, fmt.Errorf("failed to get key: %v", err)
		}
	}

	el, err := openpgp.ReadArmoredKeyRing(strings.NewReader(keyText))
	if err != nil {
		return nil, err
	}
	if len(el) == 0 {
		return nil, fmt.Errorf("no keys in keyring")
	}
	if len(el) > 1 {
		return nil, fmt.Errorf("server returned more than one key for unique fingerprint")
	}
	return el, nil
}

func serializeEntity(e *openpgp.Entity, blockType string) (string, error) {
	w := bytes.NewBuffer(nil)

	wr, err := armor.Encode(w, blockType, nil)
	if err != nil {
		return "", err
	}

	if err = e.Serialize(wr); err != nil {
		wr.Close()
		return "", err
	}
	wr.Close()

	return w.String(), nil
}

func serializePrivateEntity(e *openpgp.Entity, blockType string) (string, error) {
	w := bytes.NewBuffer(nil)

	wr, err := armor.Encode(w, blockType, nil)
	if err != nil {
		return "", err
	}

	if err = e.SerializePrivate(wr, nil); err != nil {
		wr.Close()
		return "", err
	}
	wr.Close()

	return w.String(), nil
}

// RecryptKey Will decrypt a entity, then recrypt it with the same password.
// This function seems pritty usless, but its not!
func RecryptKey(k *openpgp.Entity, passphrase []byte) error {
	if !k.PrivateKey.Encrypted {
		return errNotEncrypted
	}

	if err := k.PrivateKey.Decrypt(passphrase); err != nil {
		return err
	}

	if err := k.PrivateKey.Encrypt(passphrase); err != nil {
		return err
	}

	return nil
}

// ExportPrivateKey Will export a private key into a file (kpath).
func (keyring *Handle) ExportPrivateKey(kpath string, armor bool) error {
	if err := keyring.PathsCheck(); err != nil {
		return err
	}

	localEntityList, err := loadKeyring(keyring.SecretPath())
	if err != nil {
		return fmt.Errorf("unable to load private keyring: %v", err)
	}

	// Get a entity to export
	entityToExport, err := SelectPrivKey(localEntityList)
	if err != nil {
		return err
	}

	if entityToExport.PrivateKey.Encrypted {
		pass, err := interactive.AskQuestionNoEcho("Enter key passphrase : ")
		if err != nil {
			return err
		}
		err = RecryptKey(entityToExport, []byte(pass))
		if err != nil {
			return err
		}
	}

	// Create the file that we will be exporting to
	file, err := os.Create(kpath)
	if err != nil {
		return err
	}
	defer file.Close()

	if !armor {
		// Export the key to the file
		err = entityToExport.SerializePrivate(file, nil)
	} else {
		var keyText string
		keyText, err = serializePrivateEntity(entityToExport, openpgp.PrivateKeyType)
		if err != nil {
			return fmt.Errorf("failed to read ASCII key format: %s", err)
		}
		file.WriteString(keyText)
	}

	if err != nil {
		return fmt.Errorf("unable to serialize private key: %v", err)
	}
	fmt.Printf("Private key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, kpath)

	return nil
}

// ExportPubKey Will export a public key into a file (kpath).
func (keyring *Handle) ExportPubKey(kpath string, armor bool) error {
	if err := keyring.PathsCheck(); err != nil {
		return err
	}

	localEntityList, err := loadKeyring(keyring.PublicPath())
	if err != nil {
		return fmt.Errorf("unable to open local keyring: %v", err)
	}

	entityToExport, err := selectPubKey(localEntityList)
	if err != nil {
		return err
	}

	file, err := os.Create(kpath)
	if err != nil {
		return fmt.Errorf("unable to create file: %v", err)
	}
	defer file.Close()

	if armor {
		var keyText string
		keyText, err = serializeEntity(entityToExport, openpgp.PublicKeyType)
		file.WriteString(keyText)
	} else {
		err = entityToExport.Serialize(file)
	}

	if err != nil {
		return fmt.Errorf("unable to serialize public key: %v", err)
	}
	fmt.Printf("Public key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, kpath)

	return nil
}

func findEntityByFingerprint(entities openpgp.EntityList, fingerprint [20]byte) *openpgp.Entity {
	for _, entity := range entities {
		if entity.PrimaryKey.Fingerprint == fingerprint {
			return entity
		}
	}

	return nil
}

// importPrivateKey imports the specified openpgp Entity, which should
// represent a private key. The entity is added to the private keyring.
func (keyring *Handle) importPrivateKey(entity *openpgp.Entity, setNewPassword bool) error {
	if entity.PrivateKey == nil {
		return fmt.Errorf("corrupted key, unable to recover data")
	}

	// Load the local private keys as entitylist
	privateEntityList, err := keyring.LoadPrivKeyring()
	if err != nil {
		return err
	}

	if findEntityByFingerprint(privateEntityList, entity.PrimaryKey.Fingerprint) != nil {
		return &KeyExistsError{fingerprint: entity.PrivateKey.Fingerprint}
	}

	newEntity := *entity

	var password string
	if entity.PrivateKey.Encrypted {
		password, err = interactive.AskQuestionNoEcho("Enter your key password : ")
		if err != nil {
			return err
		}
		if err := newEntity.PrivateKey.Decrypt([]byte(password)); err != nil {
			return err
		}
	}

	if setNewPassword {
		// Get a new password for the key
		password, err = interactive.GetPassphrase("Enter a new password for this key : ", 3)
		if err != nil {
			return err
		}
	}

	if password != "" {
		if err := newEntity.PrivateKey.Encrypt([]byte(password)); err != nil {
			return err
		}
	}

	// Store the private key
	if err := keyring.appendPrivateKey(&newEntity); err != nil {
		return err
	}

	return nil
}

// importPublicKey imports the specified openpgp Entity, which should
// represent a public key. The entity is added to the public keyring.
func (keyring *Handle) importPublicKey(entity *openpgp.Entity) error {
	// Load the local public keys as entitylist
	publicEntityList, err := keyring.LoadPubKeyring()
	if err != nil {
		return err
	}

	if findEntityByFingerprint(publicEntityList, entity.PrimaryKey.Fingerprint) != nil {
		return &KeyExistsError{fingerprint: entity.PrimaryKey.Fingerprint}
	}

	if err := keyring.appendPubKey(entity); err != nil {
		return err
	}

	return nil
}

// ImportKey imports one or more keys from the specified file. The keys
// can be either a public or private keys, and the file can be either in
// binary or ascii-armored format.
func (keyring *Handle) ImportKey(kpath string, setNewPassword bool) error {
	// Load the private key as an entitylist
	pathEntityList, err := loadKeysFromFile(kpath)
	if err != nil {
		return fmt.Errorf("unable to get entity from: %s: %v", kpath, err)
	}

	for _, pathEntity := range pathEntityList {
		if pathEntity.PrivateKey != nil {
			// We have a private key
			err := keyring.importPrivateKey(pathEntity, setNewPassword)
			if err != nil {
				return err
			}

			fmt.Printf("Key with fingerprint %X successfully added to the private keyring\n",
				pathEntity.PrivateKey.Fingerprint)
		}

		// There's no else here because a single entity can have
		// both a private and public keys
		if pathEntity.PrimaryKey != nil {
			// We have a public key
			err := keyring.importPublicKey(pathEntity)
			if err != nil {
				return err
			}

			fmt.Printf("Key with fingerprint %X successfully added to the public keyring\n",
				pathEntity.PrimaryKey.Fingerprint)
		}
	}

	return nil
}

// PushPubkey pushes a public key to the Key Service.
func PushPubkey(httpClient *http.Client, e *openpgp.Entity, keyserverURI, authToken string) error {
	keyText, err := serializeEntity(e, openpgp.PublicKeyType)
	if err != nil {
		return err
	}

	// Get a Key Service client.
	c, err := client.NewClient(&client.Config{
		BaseURL:    keyserverURI,
		AuthToken:  authToken,
		HTTPClient: httpClient,
	})
	if err != nil {
		return err
	}

	// Push key to Key Service.
	if err := c.PKSAdd(context.TODO(), keyText); err != nil {
		if jerr, ok := err.(*jsonresp.Error); ok && jerr.Code == http.StatusUnauthorized {
			// The request failed with HTTP code unauthorized. Guide user to fix that.
			sylog.Infof(helpAuth+helpPush, e.PrimaryKey.Fingerprint)
			return fmt.Errorf("unauthorized or missing token")
		}
		return fmt.Errorf("key server did not accept PGP key: %v", err)

	}
	return nil
}
