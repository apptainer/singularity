// Copyright (c) 2018-2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

// Package sypgp implements the openpgp integration into the singularity project.
package sypgp

import (
	"bufio"
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

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/pkg/syfs"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ssh/terminal"
)

const helpAuth = `Access token is expired or missing. To update or obtain a token:
  1) View configured remotes using "singularity remote list"
  2) Identify default remote. It will be listed with square brackets.
  3) Login to default remote with "singularity remote login <RemoteName>"
`
const helpPush = `  4) Push key using "singularity key push %[1]X"
`

var errPassphraseMismatch = errors.New("passphrases do not match")
var errTooManyRetries = errors.New("too many retries while getting a passphrase")
var errNotEncrypted = errors.New("key is not encrypted")

// KeyExistsError is a type representing an error associated to a specific key.
type KeyExistsError struct {
	fingerprint [20]byte
}

func (e *KeyExistsError) Error() string {
	return fmt.Sprintf("the key with fingerprint %X already belongs to the keyring", e.fingerprint)
}

// askQuestionUsingGenericDescr reads from a file descriptor (more precisely
// from a *os.File object) one line at a time. The file can be a normal file or
// os.Stdin.
// Note that we could imagine a simpler code but we want to make sure that the
// code works properly in the normal case with the default Stdin and when
// redirecting stdin (for testing or when using pipes).
//
// TODO: use a io.ReadSeeker instead of a *os.File
func askQuestionUsingGenericDescr(f *os.File) (string, error) {
	// Get the initial position in the buffer so we can later seek the correct
	// position based on how much data we read. Doing so, we can still benefit
	// from buffered IO and still have a fine-grain controlover reading
	// operations.
	// Note that we do not check for errirs since some cases (e.g., pipes) will
	// actually not allow to perform a seek. This is intended and basically a
	// no-op in that context.
	pos, _ := f.Seek(0, os.SEEK_CUR)
	// Get the data
	scanner := bufio.NewScanner(f)
	tok := scanner.Scan()
	if !tok {
		return "", scanner.Err()
	}
	response := scanner.Text()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	// We did a buffered read (for good reasons, it is generic), so we make
	// sure we reposition ourselves at the end of the data that was read, not
	// the end of the buffer, so we can make sure that we read the data line
	// by line and do not drop data after a lot more data was read from the
	// file descriptor. In other terms, we may have read a very small subset
	// of the available data and make sure we reposition ourselves at the
	// end of the data we handled, not at the end of the data that was read
	// from the file descriptor.
	strLen := 1 // We always move forward, even if we get an empty response
	if len(response) > 1 {
		strLen += len(response)
	}
	// Note that we do not check for errors since some cases (e.g., pipes)
	// will actually not allow to perform a Seek(). This is intended and
	// will not create a problem.
	f.Seek(pos+int64(strLen), os.SEEK_SET)

	return response, nil
}

// AskQuestion prompts the user with a question and return the response
func AskQuestion(format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)
	return askQuestionUsingGenericDescr(os.Stdin)
}

// askYNQuestion prompts the user expecting an answer that's either "y",
// "n" or a blank, in which case defaultAnswer is returned.
func askYNQuestion(defaultAnswer, format string, a ...interface{}) (string, error) {
	ans, err := AskQuestion(format, a...)
	if err != nil {
		return "", err
	}

	switch ans := strings.ToLower(ans); ans {
	case "y", "yes":
		return "y", nil

	case "n", "no":
		return "n", nil

	case "":
		return defaultAnswer, nil

	default:
		return "", fmt.Errorf("invalid answer %q", ans)
	}
}

// AskQuestionNoEcho works like AskQuestion() except it doesn't echo user's input
func AskQuestionNoEcho(format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)

	var response string
	var err error
	// Go provides a package for handling terminal and more specifically
	// reading password from terminal. We want to use the package when possible
	// since it gives us an easy and secure way to interactively get the
	// password from the user. However, this is only working when the
	// underlying file descriptor is associated to a VT100 terminal, not with
	// other file descriptors, including when redirecting Stdin to an actual
	// file in the context of testing or in the context of pipes.
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		var resp []byte
		resp, err = terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", err
		}
		response = string(resp)
	} else {
		response, err = askQuestionUsingGenericDescr(os.Stdin)
		if err != nil {
			return "", err
		}
	}
	fmt.Println("")
	return string(response), nil
}

// GetTokenFile returns a string describing the path to the stored token file
func GetTokenFile() string {
	return filepath.Join(syfs.ConfigDir(), "sylabs-token")
}

// DirPath returns a string describing the path to the sypgp home folder
func DirPath() string {
	sypgpDir := os.Getenv("SINGULARITY_SYPGPDIR")
	if sypgpDir == "" {
		return filepath.Join(syfs.ConfigDir(), "sypgp")
	}
	return sypgpDir
}

// SecretPath returns a string describing the path to the private keys store
func SecretPath() string {
	return filepath.Join(DirPath(), "pgp-secret")
}

// PublicPath returns a string describing the path to the public keys store
func PublicPath() string {
	return filepath.Join(DirPath(), "pgp-public")
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
func PathsCheck() error {
	if err := ensureDirPrivate(DirPath()); err != nil {
		return err
	}

	if err := ensureFilePrivate(SecretPath()); err != nil {
		return err
	}

	if err := ensureFilePrivate(PublicPath()); err != nil {
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
func LoadPrivKeyring() (openpgp.EntityList, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	return loadKeyring(SecretPath())
}

// LoadPubKeyring loads the public keys from local store into an EntityList
func LoadPubKeyring() (openpgp.EntityList, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	return loadKeyring(PublicPath())
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
func PrintPubKeyring() error {
	pubEntlist, err := LoadPubKeyring()
	if err != nil {
		return err
	}

	printEntities(os.Stdout, pubEntlist)

	return nil
}

// PrintPrivKeyring prints the secret keyring read from the public local store
func PrintPrivKeyring() error {
	privEntlist, err := LoadPrivKeyring()
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
func appendPrivateKey(e *openpgp.Entity) error {
	f, err := createOrAppendPrivateFile(SecretPath())
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
func appendPubKey(e *openpgp.Entity) error {
	f, err := createOrAppendPrivateFile(PublicPath())
	if err != nil {
		return err
	}
	defer f.Close()

	return storePubKeys(f, openpgp.EntityList{e})
}

// storePubKeyring overwrites the public keyring with the listed keys
func storePubKeyring(keys openpgp.EntityList) error {
	f, err := os.OpenFile(PublicPath(), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
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
func CheckLocalPubKey(ckey string) (bool, error) {
	// read all the local public keys
	elist, err := loadKeyring(PublicPath())
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
func RemovePubKey(toDelete string) error {
	// read all the local public keys
	elist, err := loadKeyring(PublicPath())
	switch {
	case os.IsNotExist(err):
		return nil

	case err != nil:
		return fmt.Errorf("unable to list local keyring: %v", err)
	}

	var newKeyList openpgp.EntityList

	matchKey := false

	// sort through them, and remove any that match toDelete
	for i := range elist {
		// if the elist[i] dose not match toDelete, then add it to newKeyList
		if !compareKeyEntity(elist[i], toDelete) {
			newKeyList = append(newKeyList, elist[i])
		} else {
			matchKey = true
		}
	}

	if !matchKey {
		return fmt.Errorf("no key matching given fingerprint found")
	}

	sylog.Verbosef("Updating local keyring: %v", PublicPath())

	return storePubKeyring(newKeyList)
}

// GetPassphrase will ask the user for a password with int number of
// retries.
func GetPassphrase(message string, retries int) (string, error) {
	ask := func() (string, error) {
		pass1, err := AskQuestionNoEcho(message)
		if err != nil {
			return "", err
		}

		pass2, err := AskQuestionNoEcho("Retype your passphrase : ")
		if err != nil {
			return "", err
		}

		if pass1 != pass2 {
			return "", errPassphraseMismatch
		}

		return pass1, nil
	}

	for i := 0; i < retries; i++ {
		switch passphrase, err := ask(); err {
		case nil:
			// we got it!
			return passphrase, nil
		case errPassphraseMismatch:
			// retry
			sylog.Warningf("%v", err)
		default:
			// something else went wrong, bail out
			return "", err
		}
	}

	return "", errTooManyRetries
}

func genKeyPair(name, comment, email, passphrase string) (*openpgp.Entity, error) {
	conf := &packet.Config{RSABits: 4096, DefaultHash: crypto.SHA384}

	entity, err := openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		return nil, err
	}

	// Encrypt private key
	if err = EncryptKey(entity, passphrase); err != nil {
		return nil, err
	}

	// Store key parts in local key caches
	if err = appendPrivateKey(entity); err != nil {
		return nil, err
	}

	if err = appendPubKey(entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// GenKeyPair generates an PGP key pair and store them in the sypgp home folder
func GenKeyPair(keyServiceURI string, authToken string) (*openpgp.Entity, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	name, err := AskQuestion("Enter your name (e.g., John Doe) : ")
	if err != nil {
		return nil, err
	}

	email, err := AskQuestion("Enter your email address (e.g., john.doe@example.com) : ")
	if err != nil {
		return nil, err
	}

	comment, err := AskQuestion("Enter optional comment (e.g., development keys) : ")
	if err != nil {
		return nil, err
	}

	// get a password
	passphrase, err := GetPassphrase("Enter a passphrase : ", 3)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Generating Entity and OpenPGP Key Pair... ")

	entity, err := genKeyPair(name, comment, email, passphrase)
	if err != nil {
		return nil, err
	}

	fmt.Printf("done\n")

	// Ask to push the new key to the keystore
	ans, err := askYNQuestion("y", "Would you like to push it to the keystore? [Y,n] ")
	switch {
	case err != nil:
		fmt.Fprintf(os.Stderr, "Not pushing newly created key to keystore: %s\n", err)

	case ans == "y":
		err = PushPubkey(entity, keyServiceURI, authToken)
		if err != nil {
			fmt.Printf("Failed to push newly created key to keystore: %s\n", err)
		} else {
			fmt.Printf("Key successfully pushed to: %s\n", keyServiceURI)
		}

	default:
		fmt.Printf("NOT pushing newly created key to: %s\n", keyServiceURI)
	}

	return entity, nil
}

// DecryptKey decrypts a private key provided a pass phrase
func DecryptKey(k *openpgp.Entity, message string) error {
	if !k.PrivateKey.Encrypted {
		return errNotEncrypted
	}

	if message == "" {
		message = "Enter key passphrase : "
	}

	pass, err := AskQuestionNoEcho(message)
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

// SelectPubKey prints a public key list to user and returns the choice
func SelectPubKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	PrintPubKeyring()

	index, err := AskQuestion("Enter # of public key to use : ")
	if err != nil {
		return nil, err
	}
	if index == "" {
		return nil, fmt.Errorf("invalid key choice")
	}
	i, err := strconv.ParseUint(index, 10, 32)
	if err != nil {
		return nil, err
	}

	if i < 0 || i > uint64(len(el))-1 {
		return nil, fmt.Errorf("invalid key choice")
	}

	return el[i], nil
}

// SelectPrivKey prints a secret key list to user and returns the choice
func SelectPrivKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	PrintPrivKeyring()

	index, err := AskQuestion("Enter # of signing key to use : ")
	if err != nil {
		return nil, err
	}
	if index == "" {
		return nil, fmt.Errorf("invalid key choice")
	}
	i, err := strconv.ParseUint(index, 10, 32)
	if err != nil {
		return nil, err
	}

	if i < 0 || i > uint64(len(el))-1 {
		return nil, fmt.Errorf("invalid key choice")
	}

	return el[i], nil
}

// SearchPubkey connects to a key server and searches for a specific key
func SearchPubkey(search, keyserverURI, authToken string) error {

	// Get a Key Service client.
	c, err := client.NewClient(&client.Config{
		BaseURL:   keyserverURI,
		AuthToken: authToken,
	})
	if err != nil {
		return err
	}

	// the max entities to print.
	pd := client.PageDetails{
		// still will only print 100 entities
		Size: 256,
	}

	// Retrieve first page of search results from Key Service.
	keyText, err := c.PKSLookup(context.TODO(), &pd, search, client.OperationIndex, true, false, nil)
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

	fmt.Printf("%v", keyText)

	return nil
}

// FetchPubkey pulls a public key from the Key Service.
func FetchPubkey(fingerprint, keyserverURI, authToken string, noPrompt bool) (openpgp.EntityList, error) {

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
		BaseURL:   keyserverURI,
		AuthToken: authToken,
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
func RecryptKey(k *openpgp.Entity) error {
	if k.PrivateKey.Encrypted {
		pass, err := AskQuestionNoEcho("Enter key passphrase : ")
		if err != nil {
			return err
		}
		err = k.PrivateKey.Decrypt([]byte(pass))
		if err != nil {
			return err
		}
		err = k.PrivateKey.Encrypt([]byte(pass))
		if err != nil {
			return err
		}
	}

	return nil
}

// ExportPrivateKey Will export a private key into a file (kpath).
func ExportPrivateKey(kpath string, armor bool) error {
	localEntityList, err := loadKeyring(SecretPath())
	if err != nil {
		return fmt.Errorf("unable to load private keyring: %v", err)
	}

	// Get a entity to export
	entityToExport, err := SelectPrivKey(localEntityList)
	if err != nil {
		return err
	}

	err = RecryptKey(entityToExport)
	if err != nil {
		return err
	}

	// Create the file that we will be exporting to
	file, err := os.Create(kpath)
	if err != nil {
		return err
	}

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
	defer file.Close()

	if err != nil {
		return fmt.Errorf("unable to serialize private key: %v", err)
	}
	fmt.Printf("Private key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, kpath)

	return nil
}

// ExportPubKey Will export a public key into a file (kpath).
func ExportPubKey(kpath string, armor bool) error {
	localEntityList, err := loadKeyring(PublicPath())
	if err != nil {
		return fmt.Errorf("unable to open local keyring: %v", err)
	}

	entityToExport, err := SelectPubKey(localEntityList)
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
func importPrivateKey(entity *openpgp.Entity) error {
	// Load the local private keys as entitylist
	privateEntityList, err := LoadPrivKeyring()
	if err != nil {
		return err
	}

	if findEntityByFingerprint(privateEntityList, entity.PrimaryKey.Fingerprint) != nil {
		return &KeyExistsError{fingerprint: entity.PrivateKey.Fingerprint}
	}

	// Check if the key is encrypted, if it is, decrypt it
	if entity.PrivateKey == nil {
		return fmt.Errorf("corrupted key, unable to recover data")
	}

	// Make a clone of the entity
	newEntity := &openpgp.Entity{
		PrimaryKey:  entity.PrimaryKey,
		PrivateKey:  entity.PrivateKey,
		Identities:  entity.Identities,
		Revocations: entity.Revocations,
		Subkeys:     entity.Subkeys,
	}

	if entity.PrivateKey.Encrypted {
		if err := DecryptKey(newEntity, "Enter your old password : "); err != nil {
			return err
		}
	}

	// Get a new password for the key
	newPass, err := GetPassphrase("Enter a new password for this key : ", 3)
	if err != nil {
		return err
	}

	if err := EncryptKey(newEntity, newPass); err != nil {
		return err
	}

	// Store the private key
	if err := appendPrivateKey(newEntity); err != nil {
		return err
	}

	return nil
}

// importPublicKey imports the specified openpgp Entity, which should
// represent a public key. The entity is added to the public keyring.
func importPublicKey(entity *openpgp.Entity) error {
	// Load the local public keys as entitylist
	publicEntityList, err := LoadPubKeyring()
	if err != nil {
		return err
	}

	if findEntityByFingerprint(publicEntityList, entity.PrimaryKey.Fingerprint) != nil {
		return &KeyExistsError{fingerprint: entity.PrimaryKey.Fingerprint}
	}

	if err := appendPubKey(entity); err != nil {
		return err
	}

	return nil
}

// ImportKey imports one or more keys from the specified file. The keys
// can be either a public or private keys, and the file can be either in
// binary or ascii-armored format.
func ImportKey(kpath string) error {
	// Load the private key as an entitylist
	pathEntityList, err := loadKeysFromFile(kpath)
	if err != nil {
		return fmt.Errorf("unable to get entity from: %s: %v", kpath, err)
	}

	for _, pathEntity := range pathEntityList {
		if pathEntity.PrivateKey != nil {
			// We have a private key
			err := importPrivateKey(pathEntity)
			if err != nil {
				return err
			}

			fmt.Printf("Key with fingerprint %X succesfully added to the private keyring\n",
				pathEntity.PrivateKey.Fingerprint)
		}

		// There's no else here because a single entity can have
		// both a private and public keys
		if pathEntity.PrimaryKey != nil {
			// We have a public key
			err := importPublicKey(pathEntity)
			if err != nil {
				return err
			}

			fmt.Printf("Key with fingerprint %X succesfully added to the public keyring\n",
				pathEntity.PrimaryKey.Fingerprint)
		}
	}

	return nil
}

// PushPubkey pushes a public key to the Key Service.
func PushPubkey(e *openpgp.Entity, keyserverURI, authToken string) error {
	keyText, err := serializeEntity(e, openpgp.PublicKeyType)
	if err != nil {
		return err
	}

	// Get a Key Service client.
	c, err := client.NewClient(&client.Config{
		BaseURL:   keyserverURI,
		AuthToken: authToken,
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
