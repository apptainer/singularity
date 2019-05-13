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
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ssh/terminal"
)

// PublicKeyType is the armor type for a PGP public key.
const PublicKeyType = "PGP PUBLIC KEY BLOCK"

// PrivateKeyType is the armor type for a PGP private key.
const PrivateKeyType = "PGP PRIVATE KEY BLOCK"

const helpAuth = `Access token is expired or missing. To update or obtain a token:
  1) View configured remotes using "singularity remote list"
  2) Identify default remote. It will be listed with square brackets.
  3) Login to default remote with "singularity remote login <RemoteName>"
`
const helpPush = `  4) Push key using "singularity key push %[1]X"
`

var errPassphraseMismatch = errors.New("passphrases do not match")
var errTooManyRetries = errors.New("too many retries while getting a passphrase")

// AskQuestion prompts the user with a question and return the response
func AskQuestion(format string, a ...interface{}) (string, error) {
	fmt.Printf(format, a...)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := scanner.Text()
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return response, nil
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
	response, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println("")
	if err != nil {
		return "", err
	}
	return string(response), nil
}

// getSingularityDir returns the directory where the user's singularity
// configuration and data is located.
func getSingularityDir() string {
	user, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		sylog.Warningf("Could not lookup user's real home directory: %s", err)

		cwd, err := os.Getwd()
		if err != nil {
			sylog.Warningf("Could not get current working directory: %s", err)
			return ".singularity"
		}

		dir := filepath.Join(cwd, ".singularity")
		sylog.Warningf("Using current directory: %s", dir)
		return dir
	}

	return filepath.Join(user.Dir, ".singularity")
}

// GetTokenFile returns a string describing the path to the stored token file
func GetTokenFile() string {
	return filepath.Join(getSingularityDir(), "sylabs-token")
}

// DirPath returns a string describing the path to the sypgp home folder
func DirPath() string {
	return filepath.Join(getSingularityDir(), "sypgp")
}

// SecretPath returns a string describing the path to the private keys store
func SecretPath() string {
	return filepath.Join(DirPath(), "pgp-secret")
}

// PublicPath returns a string describing the path to the public keys store
func PublicPath() string {
	return filepath.Join(DirPath(), "pgp-public")
}

func ensureDirPrivate(dn string) error {
	mode := os.FileMode(0700)

	oldumask := syscall.Umask(0077)

	err := os.MkdirAll(dn, mode)

	// restore umask...
	syscall.Umask(oldumask)

	// ... and check if there was an error

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

// LoadPrivKeyring loads the private keys from local store into an EntityList
func LoadPrivKeyring() (openpgp.EntityList, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	f, err := os.Open(SecretPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return openpgp.ReadKeyRing(f)
}

// LoadPubKeyring loads the public keys from local store into an EntityList
func LoadPubKeyring() (openpgp.EntityList, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	f, err := os.Open(PublicPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return openpgp.ReadKeyRing(f)
}

// PrintEntity pretty prints an entity entry
func PrintEntity(index int, e *openpgp.Entity) {
	for _, v := range e.Identities {
		fmt.Printf("%v) U: %v (%v) <%v>\n", index, v.UserId.Name, v.UserId.Comment, v.UserId.Email)
	}
	fmt.Printf("   C: %v\n", e.PrimaryKey.CreationTime)
	fmt.Printf("   F: %0X\n", e.PrimaryKey.Fingerprint)
	bits, _ := e.PrimaryKey.BitLength()
	fmt.Printf("   L: %v\n", bits)
}

// PrintPubKeyring prints the public keyring read from the public local store
func PrintPubKeyring() (err error) {
	var pubEntlist openpgp.EntityList

	if pubEntlist, err = LoadPubKeyring(); err != nil {
		return
	}

	for i, e := range pubEntlist {
		PrintEntity(i, e)
		fmt.Println("   --------")
	}

	return
}

// PrintPrivKeyring prints the secret keyring read from the public local store
func PrintPrivKeyring() (err error) {
	var privEntlist openpgp.EntityList

	if privEntlist, err = LoadPrivKeyring(); err != nil {
		return
	}

	for i, e := range privEntlist {
		PrintEntity(i, e)
		fmt.Println("   --------")
	}

	return
}

// StorePrivKey stores a private entity list into the local key cache
func StorePrivKey(e *openpgp.Entity) (err error) {
	f, err := os.OpenFile(SecretPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	if err = e.SerializePrivate(f, nil); err != nil {
		return
	}
	return
}

// StorePubKey stores a public key entity list into the local key cache
func StorePubKey(e *openpgp.Entity) (err error) {
	f, err := os.OpenFile(PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	if err = e.Serialize(f); err != nil {
		return
	}
	return
}

// CompareKeyEntity compares a key ID with a string, returning true if the
// key and oldToken match.
func CompareKeyEntity(e *openpgp.Entity, oldToken string) bool {
	// TODO: there must be a better way to do this...
	return fmt.Sprintf("%X", e.PrimaryKey.Fingerprint) == oldToken
}

// CheckLocalPubKey will check if we have a local public key matching ckey string
// returns true if there's a match.
func CheckLocalPubKey(ckey string) (bool, error) {
	f, err := os.OpenFile(PublicPath(), os.O_CREATE|os.O_RDONLY, 0600)
	if err != nil {
		return false, fmt.Errorf("unable to open local keyring: %v", err)
	}
	defer f.Close()

	// read all the local public keys
	elist, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return false, fmt.Errorf("unable to list local keyring: %v", err)
	}

	for i := range elist {
		if CompareKeyEntity(elist[i], ckey) {
			return true, nil
		}
	}
	return false, nil
}

// RemovePubKey will delete a public key matching toDelete
func RemovePubKey(toDelete string) error {
	f, err := os.OpenFile(PublicPath(), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("unable to open local keyring: %v", err)
	}
	defer f.Close()

	// read all the local public keys
	elist, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return fmt.Errorf("unable to list local keyring: %v", err)
	}

	var newKeyList []openpgp.Entity

	matchKey := false

	// sort through them, and remove any that match toDelete
	for i := range elist {
		// if the elist[i] dose not match toDelete, then add it to newKeyList
		if !CompareKeyEntity(elist[i], toDelete) {
			newKeyList = append(newKeyList, *elist[i])
		} else {
			matchKey = true
		}
	}

	if !matchKey {
		return fmt.Errorf("no key matching given fingerprint found")
	}

	sylog.Verbosef("Updating local keyring: %v", PublicPath())

	// open the public keyring file
	nf, err := os.OpenFile(PublicPath(), os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to clear, and open the file: %v", err)
	}
	defer nf.Close()

	// loop through a write all the other keys back
	for k := range newKeyList {
		// store the keys
		if err := StorePubKey(&newKeyList[k]); err != nil {
			return fmt.Errorf("could not store public key: %s", err)
		}
	}
	return nil
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

// GenKeyPair generates an OpenPGP key pair and store them in the sypgp home folder
func GenKeyPair(keyServiceURI string, authToken string) (entity *openpgp.Entity, err error) {
	conf := &packet.Config{RSABits: 4096, DefaultHash: crypto.SHA384}

	if err = PathsCheck(); err != nil {
		return
	}

	name, err := AskQuestion("Enter your name (e.g., John Doe) : ")
	if err != nil {
		return
	}

	email, err := AskQuestion("Enter your email address (e.g., john.doe@example.com) : ")
	if err != nil {
		return
	}

	comment, err := AskQuestion("Enter optional comment (e.g., development keys) : ")
	if err != nil {
		return
	}

	// get a password
	passphrase, err := GetPassphrase("Enter a passphrase : ", 3)
	if err != nil {
		return
	}

	fmt.Printf("Generating Entity and OpenPGP Key Pair...")
	entity, err = openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		return
	}
	fmt.Printf("done\n")

	// encrypt private key
	if err = EncryptKey(entity, passphrase); err != nil {
		return
	}

	// Store key parts in local key caches
	if err = StorePrivKey(entity); err != nil {
		return
	}
	if err = StorePubKey(entity); err != nil {
		return
	}

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
	if message == "" {
		message = "Enter key passphrase : "
	}
	if k.PrivateKey.Encrypted {
		pass, err := AskQuestionNoEcho(message)
		if err != nil {
			return err
		}

		if err := k.PrivateKey.Decrypt([]byte(pass)); err != nil {
			return err
		}
	}
	return nil
}

// EncryptKey encrypts a private key using a pass phrase
func EncryptKey(k *openpgp.Entity, pass string) (err error) {
	if k.PrivateKey.Encrypted {
		return fmt.Errorf("key already encrypted")
	}
	err = k.PrivateKey.Encrypt([]byte(pass))
	return
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

func ReformatGPGExportedFile(r io.Reader) io.Reader {

	var keyString string
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)

	s := buf.String()

	//remove trailing line at the EOF if present, otherwise return the same content
	if s[len(s)-1] == '\n' {
		keyString = s[:len(s)-1]
	} else {
		keyString = s[:]
	}
	//add missing part of header
	if keyString[0:5] != "-----" {
		keyString = "--" + keyString
	}

	return strings.NewReader(keyString)
}

// LoadKeyringFromFile loads a key from a local file (private or public) given from a path into an EntityList
func LoadKeyringFromFile(path string) (openpgp.EntityList, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		reader := ReformatGPGExportedFile(f)
		return openpgp.ReadArmoredKeyRing(reader)
	}
	return el, err

}

// ExportPrivateKey Will export a private key into a file (kpath).
func ExportPrivateKey(kpath string, armor bool) error {

	f, err := os.OpenFile(SecretPath(), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("unable to open local keyring: %v", err)
	}
	defer f.Close()

	localEntityList, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return fmt.Errorf("unable to list local keyring: %v", err)
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
	file, err := os.Create(kpath)
	if err != nil {
		return fmt.Errorf("unable to create file: %v", err)
	}
	f, err := os.OpenFile(PublicPath(), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("unable to open local keyring: %v", err)
	}
	defer f.Close()

	localEntityList, err := openpgp.ReadKeyRing(f)
	if err != nil {
		return fmt.Errorf("unable to list local keyring: %v", err)
	}

	entityToExport, err := SelectPubKey(localEntityList)
	if err != nil {
		return err
	}

	if !armor {
		err = entityToExport.Serialize(file)
	} else {
		var keyText string
		keyText, err = serializeEntity(entityToExport, openpgp.PublicKeyType)
		file.WriteString(keyText)
	}

	if err != nil {
		return fmt.Errorf("unable to serialize public key: %v", err)
	}
	defer file.Close()
	fmt.Printf("Public key with fingerprint %X correctly exported to file: %s\n", entityToExport.PrimaryKey.Fingerprint, kpath)

	return nil
}

// ImportPrivateKey Will import a private key from a file (kpath).
func ImportPrivateKey(entity *openpgp.Entity) error {
	// Load the local private keys as entitylist

	privateEntityList, err := LoadPrivKeyring()
	if err != nil {
		return err
	}

	// Get local keyring (where the key will be stored)
	secretFilePath, err := os.OpenFile(SecretPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer secretFilePath.Close()

	isInStore := false

	for _, privateEntity := range privateEntityList {
		if privateEntity.PrimaryKey.Fingerprint == entity.PrimaryKey.Fingerprint {
			isInStore = true
			break
		}

	}

	if !isInStore {
		// Make a clone of the entity
		newEntity := &openpgp.Entity{
			PrimaryKey:  entity.PrimaryKey,
			PrivateKey:  entity.PrivateKey,
			Identities:  entity.Identities,
			Revocations: entity.Revocations,
			Subkeys:     entity.Subkeys,
		}

		// Check if the key is encrypted, if it is, decrypt it
		if entity.PrivateKey != nil {
			if entity.PrivateKey.Encrypted {
				err := DecryptKey(newEntity, "Enter your old password : ")
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("corrupted key, unable to recover data")
		}

		// Get a new password for the key
		newPass, err := GetPassphrase("Enter a new password for this key : ", 3)
		if err != nil {
			return err
		}
		err = EncryptKey(newEntity, newPass)
		if err != nil {
			return err
		}

		// Store the private key
		err = StorePrivKey(newEntity)
		if err != nil {
			return err
		}
		fmt.Printf("Key with fingerprint %X succesfully added to the keyring\n", entity.PrimaryKey.Fingerprint)
	} else {
		fmt.Printf("The key you want to add with fingerprint %X already belongs to the keyring\n", entity.PrimaryKey.Fingerprint)
	}
	return nil
}

// ImportPubKey Will import a public key from a file (kpath).
func ImportPubKey(entity *openpgp.Entity) error {

	// Load the local public keys as entitylist
	publicEntityList, err := LoadPubKeyring()
	if err != nil {
		return err
	}

	// Get local keystore (where the key will be stored)
	publicFilePath, err := os.OpenFile(PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer publicFilePath.Close()

	isInStore := false
	for _, publicEntity := range publicEntityList {
		if entity.PrimaryKey.KeyId == publicEntity.PrimaryKey.KeyId {
			isInStore = true
			// Verify that this key has already been added
			break
		}
	}
	if !isInStore {
		if err = entity.Serialize(publicFilePath); err != nil {
			return err
		}
		fmt.Printf("Key with fingerprint %X succesfully added to the keyring\n", entity.PrimaryKey.Fingerprint)
	} else {
		fmt.Printf("The key you want to add with fingerprint %X already belongs to the keyring\n", entity.PrimaryKey.Fingerprint)
	}

	return nil
}

func getTypesFromEntity(path string) []string {
	var types []string

	f, err := os.Open(path)
	if err != nil {
		return types
	}
	defer f.Close()

	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		// is armored, so need to identify each of the block types and store them
		re := ReformatGPGExportedFile(f)
		block, err := armor.Decode(re)
		if err != nil {
			return types
		}
		types = append(types, block.Type)
	}
	// is not armored so obtain the types checking the privatekey field from entity
	for _, pathEntity := range el {
		if pathEntity.PrivateKey != nil {
			types = append(types, PrivateKeyType)
		} else {
			types = append(types, PublicKeyType)
		}
	}

	return types
}

// ImportKey Will import a key from a file, and decied if its
// a public, or private key.
func ImportKey(kpath string) error {

	// Load the private key as an entitylist
	pathEntityList, err := LoadKeyringFromFile(kpath)
	if err != nil {
		return fmt.Errorf("unable to get entity from: %s: %v", kpath, err)
	}

	pathEntityTypes := getTypesFromEntity(kpath)

	for i, pathEntity := range pathEntityList {

		if pathEntityTypes[i] == PrivateKeyType {
			// Its a private key
			err := ImportPrivateKey(pathEntity)
			if err != nil {
				return err
			}

		}
		if pathEntityTypes[i] == PublicKeyType {
			err := ImportPubKey(pathEntity)
			if err != nil {
				return err
			}
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
