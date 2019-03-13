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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	jsonresp "github.com/sylabs/json-resp"
	"github.com/sylabs/scs-key-client/client"
	"github.com/sylabs/singularity/internal/pkg/sylog"
	"github.com/sylabs/singularity/internal/pkg/util/user"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ssh/terminal"
)

const helpAuth = `Access token is expired or missing. To update or obtain a token:
  1) Go to : https://cloud.sylabs.io/
  2) Click "Sign in to Sylabs" and follow the sign in steps
  3) Click on your login id (same and updated button as the Sign in one)
  4) Select "Access Tokens" from the drop down menu
  5) Click the "Manage my API tokens" button from the "Account Management" page
  6) Click "Create"
  7) Click "Copy token to Clipboard" from the "New API Token" page
  8) Paste the token string to the waiting prompt below and then press "Enter"

WARNING: this may overwrite a previous token if ~/.singularity/sylabs-token exists

`

// routine that outputs signature type (applies to vindex operation)
func printSigType(sig *packet.Signature) {
	switch sig.SigType {
	case packet.SigTypeBinary:
		fmt.Printf("sbin ")
	case packet.SigTypeText:
		fmt.Printf("stext")
	case packet.SigTypeGenericCert:
		fmt.Printf("sgenc")
	case packet.SigTypePersonaCert:
		fmt.Printf("sperc")
	case packet.SigTypeCasualCert:
		fmt.Printf("scasc")
	case packet.SigTypePositiveCert:
		fmt.Printf("sposc")
	case packet.SigTypeSubkeyBinding:
		fmt.Printf("sbind")
	case packet.SigTypePrimaryKeyBinding:
		fmt.Printf("sprib")
	case packet.SigTypeDirectSignature:
		fmt.Printf("sdirc")
	case packet.SigTypeKeyRevocation:
		fmt.Printf("skrev")
	case packet.SigTypeSubkeyRevocation:
		fmt.Printf("sbrev")
	}
}

// routine that displays signature information (applies to vindex operation)
func putSigInfo(sig *packet.Signature) {
	fmt.Print("sig  ")
	printSigType(sig)
	fmt.Print(" ")
	if sig.IssuerKeyId != nil {
		fmt.Printf("%08X ", uint32(*sig.IssuerKeyId))
	}
	y, m, d := sig.CreationTime.Date()
	fmt.Printf("%02d-%02d-%02d ", y, m, d)
}

// output all the signatures related to a key (entity) (applies to vindex
// operation).
func printSignatures(entity *openpgp.Entity) error {
	fmt.Println("=>++++++++++++++++++++++++++++++++++++++++++++++++++")

	fmt.Printf("uid  ")
	for _, i := range entity.Identities {
		fmt.Printf("%s", i.Name)
	}
	fmt.Println("")

	// Self signature and other Signatures
	for _, i := range entity.Identities {
		if i.SelfSignature != nil {
			putSigInfo(i.SelfSignature)
			fmt.Printf("--------- --------- [selfsig]\n")
		}
		for _, s := range i.Signatures {
			putSigInfo(s)
			fmt.Printf("--------- --------- ---------\n")
		}
	}

	// Revocation Signatures
	for _, s := range entity.Revocations {
		putSigInfo(s)
		fmt.Printf("--------- --------- ---------\n")
	}
	fmt.Println("")

	// Subkeys Signatures
	for _, sub := range entity.Subkeys {
		putSigInfo(sub.Sig)
		fmt.Printf("--------- --------- [%s]\n", entity.PrimaryKey.KeyIdShortString())
	}

	fmt.Println("<=++++++++++++++++++++++++++++++++++++++++++++++++++")

	return nil
}

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

// GetTokenFile returns a string describing the path to the stored token file
func GetTokenFile() string {
	user, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		sylog.Warningf("could not lookup user's real home folder %s", err)
		sylog.Warningf("using current directory for %s", filepath.Join(".singularity", "sylabs-token"))
		return filepath.Join(".singularity", "sylabs-token")
	}

	return filepath.Join(user.Dir, ".singularity", "sylabs-token")
}

// DirPath returns a string describing the path to the sypgp home folder
func DirPath() string {
	user, err := user.GetPwUID(uint32(os.Getuid()))
	if err != nil {
		sylog.Warningf("could not lookup user's real home folder %s", err)
		sylog.Warningf("using current directory for %s", filepath.Join(".singularity", "sypgp"))
		return filepath.Join(".singularity", "sypgp")
	}

	return filepath.Join(user.Dir, ".singularity", "sypgp")
}

// SecretPath returns a string describing the path to the private keys store
func SecretPath() string {
	return filepath.Join(DirPath(), "pgp-secret")
}

// PublicPath returns a string describing the path to the public keys store
func PublicPath() string {
	return filepath.Join(DirPath(), "pgp-public")
}

// PathsCheck creates the sypgp home folder, secret and public keyring files
func PathsCheck() error {
	// create the sypgp base directory
	if err := os.MkdirAll(DirPath(), 0700); err != nil {
		return err
	}

	dirinfo, err := os.Stat(DirPath())
	if err != nil {
		return err
	}
	if dirinfo.Mode() != os.ModeDir|0700 {
		sylog.Warningf("directory mode (%v) on %v needs to be 0700, fixing that...", dirinfo.Mode(), DirPath())
		if err = os.Chmod(DirPath(), 0700); err != nil {
			return err
		}
	}

	// create or open the secret OpenPGP key cache file
	fs, err := os.OpenFile(SecretPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer fs.Close()

	// check and fix permissions (secret cache file)
	fsinfo, err := fs.Stat()
	if err != nil {
		return err
	}
	if fsinfo.Mode() != 0600 {
		sylog.Warningf("file mode (%v) on %v needs to be 0600, fixing that...", fsinfo.Mode(), SecretPath())
		if err = fs.Chmod(0600); err != nil {
			return err
		}
	}

	// create or open the public OpenPGP key cache file
	fp, err := os.OpenFile(PublicPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()

	// check and fix permissions (public cache file)
	fpinfo, err := fp.Stat()
	if err != nil {
		return err
	}
	if fpinfo.Mode() != 0600 {
		sylog.Warningf("file mode (%v) on %v needs to be 0600, fixing that...", fpinfo.Mode(), PublicPath())
		if err = fp.Chmod(0600); err != nil {
			return err
		}
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

// LoadKeyringFromFile loads a key from a local file (private or public) given from a path into an EntityList
func LoadKeyringFromFile(path string) (openpgp.EntityList, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	el, err := openpgp.ReadArmoredKeyRing(f)
	if err != nil {
		return nil, err
	}

	return el, nil
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

// compareLocalPubKey compares a key ID with a string, returning true if the
// key and oldToken match.
func compareLocalPubKey(e *openpgp.Entity, oldToken string) bool {
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
		if compareLocalPubKey(elist[i], ckey) {
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

	// sort throught them, and remove any that match toDelete
	for i := range elist {
		// if the elist[i] dose not match toDelete, then add it to newKeyList
		if !compareLocalPubKey(elist[i], toDelete) {
			newKeyList = append(newKeyList, *elist[i])
		}
	}

	sylog.Infof("Updating local keyring: %v", PublicPath())

	// open the public keyring file
	nf, err := os.OpenFile(PublicPath(), os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to clear, and open the file: %v", err)
	}
	defer nf.Close()

	// loop through a write all the other keys back
	for k := range newKeyList {
		// store the freshly downloaded key
		if err := StorePubKey(&newKeyList[k]); err != nil {
			return fmt.Errorf("could not store public key: %s", err)
		}
	}
	return nil
}

// GenKeyPair generates an OpenPGP key pair and store them in the sypgp home folder
func GenKeyPair() (entity *openpgp.Entity, err error) {
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

	fmt.Print("Generating Entity and OpenPGP Key Pair... ")
	entity, err = openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		return
	}
	fmt.Println("Done")

	// encrypt private key
	pass, err := AskQuestionNoEcho("Enter encryption passphrase : ")
	if err != nil {
		return
	}
	if err = EncryptKey(entity, pass); err != nil {
		return
	}

	// Store key parts in local key caches
	if err = StorePrivKey(entity); err != nil {
		return
	}
	if err = StorePubKey(entity); err != nil {
		return
	}

	return
}

// DecryptKey decrypts a private key provided a pass phrase
func DecryptKey(k *openpgp.Entity) error {
	if k.PrivateKey.Encrypted == true {
		pass, err := AskQuestionNoEcho("Enter key passphrase: ")
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
	if k.PrivateKey.Encrypted == true {
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

// helpAuthentication advises the client on how to procure an authentication token
func helpAuthentication() (token string, err error) {
	sylog.Infof(helpAuth)

	token, err = AskQuestion("Paste Token HERE: ")
	if err != nil {
		return "", fmt.Errorf("could not read pasted token: %s", err)
	}

	// Create/Overwrite token file
	err = ioutil.WriteFile(GetTokenFile(), []byte(token), 0600)
	if err != nil {
		return "", fmt.Errorf("could not create/update token file: %s", err)
	}

	return
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

	pd := client.PageDetails{
		Size: 10,
	}

	// Retrieve first page of search results from Key Service.
	keyText, err := c.PKSLookup(context.TODO(), &pd, search, client.OperationIndex, true, false, nil)
	if err != nil {
		if err, ok := err.(*jsonresp.Error); ok && err.Code == http.StatusUnauthorized {

			// The request failed with HTTP code unauthorized. Guide user to fix that.
			authToken, err := helpAuthentication()
			if err != nil {
				return fmt.Errorf("could not obtain or install authentication token: %s", err)
			}

			// Try to pull key from Key Service again with new auth token.
			c.AuthToken = authToken
			pd = client.PageDetails{}
			if keyText, err = c.PKSLookup(context.TODO(), &pd, search, client.OperationIndex, true, false, nil); err != nil {
				return err
			}
		} else if err.Code == http.StatusNotFound {
			return fmt.Errorf("no matching keys found for fingerprint")
		} else {
			return fmt.Errorf("failed to get key: %v", err)
		}
	}

	// Print first page of search results.
	fmt.Print(keyText)

	// Retrieve 2-N pages of search results from Key Service.
	for pd.Token != "" {
		resp, err := AskQuestion("\nDisplay more results? [Y/n] ")
		if err != nil {
			return err
		}
		if resp != "" && resp != "y" && resp != "Y" {
			break
		}

		keyText, err := c.PKSLookup(context.TODO(), &pd, search, client.OperationIndex, true, false, nil)
		if err != nil {
			return err
		}

		// Print page of search results.
		fmt.Print(keyText)
	}

	return nil
}

// FetchPubkey pulls a public key from the Key Service.
func FetchPubkey(fingerprint, keyserverURI, authToken string, noPrompt bool) (openpgp.EntityList, error) {

	// Decode fingerprint and ensure proper length.
	var fp [20]byte
	b, err := hex.DecodeString(fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to decode fingerprint: %v", err)
	}
	if got, want := len(b), len(fp); got != want {
		return nil, fmt.Errorf("unexpected fingerprint length of %v (expected %v)", got, want)
	}
	copy(fp[:], b)

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
		if err, ok := err.(*jsonresp.Error); ok && err.Code == http.StatusUnauthorized {

			// The request failed with HTTP code unauthorized. Guide user to fix that.
			authToken, err := helpAuthentication()
			if err != nil {
				return nil, fmt.Errorf("could not obtain or install authentication token: %s", err)
			}

			// Try to pull key from Key Service again with new auth token.
			c.AuthToken = authToken
			if keyText, err = c.GetKey(context.TODO(), fp); err != nil {
				return nil, err
			}
		} else if err.Code == http.StatusNotFound {
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

// PushPubkey pushes a public key to the Key Service.
func PushPubkey(e *openpgp.Entity, keyserverURI, authToken string) error {
	keyText, err := serializeEntity(e, openpgp.PublicKeyType)

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
		if err, ok := err.(*jsonresp.Error); ok && err.Code == http.StatusUnauthorized {

			// The request failed with HTTP code unauthorized. Guide user to fix that.
			authToken, err := helpAuthentication()
			if err != nil {
				return fmt.Errorf("could not obtain or install authentication token: %s", err)
			}

			// Try to push key to Key Service again with new auth token.
			c.AuthToken = authToken
			if err := c.PKSAdd(context.TODO(), keyText); err != nil {
				return fmt.Errorf("key server did not accept PGP key: %v", err)
			}
		} else {
			return fmt.Errorf("key server did not accept PGP key: %v", err)
		}
	}
	return nil
}
