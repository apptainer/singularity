// Copyright (c) 2018, SyLabs, Inc. All rights reserved.
//
// This software is licensed under a 3-clause BSD license.  Please
// consult LICENSE file distributed with the sources of this project regarding
// your rights to use or distribute this software.

// Package sypgp implements the openpgp integration into the singularity project.
package sypgp

import (
	"bufio"
	"bytes"
	"crypto"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

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

// DirPath returns a string describing the path to the sypgp home folder
func DirPath() string {
	return filepath.Join(os.Getenv("HOME"), ".sypgp")
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
	if err := os.MkdirAll(DirPath(), 0700); err != nil {
		log.Println("could not create singularity PGP directory")
		return err
	}

	fs, err := os.OpenFile(SecretPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create private keyring file: ", err)
		return err
	}
	fs.Close()

	fp, err := os.OpenFile(PublicPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create public keyring file: ", err)
		return err
	}
	fp.Close()

	return nil
}

// LoadPrivKeyring loads the private keys from local store into an EntityList
func LoadPrivKeyring() (openpgp.EntityList, error) {
	if err := PathsCheck(); err != nil {
		return nil, err
	}

	f, err := os.Open(SecretPath())
	if err != nil {
		log.Println("Error trying to open secret keyring file: ", err)
		return nil, err
	}
	defer f.Close()

	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Println("Error while trying to read secret key ring: ", err)
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
		log.Println("Error trying to open public keyring file: ", err)
		return nil, err
	}
	defer f.Close()

	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Println("Error while trying to read public key ring: ", err)
		return nil, err
	}

	return el, nil
}

func printEntity(index int, e *openpgp.Entity) {
	for _, v := range e.Identities {
		fmt.Printf("%v) U: %v %v %v\n", index, v.UserId.Name, v.UserId.Comment, v.UserId.Email)
	}
	fmt.Printf("   C: %v\n", e.PrimaryKey.CreationTime)
	fmt.Printf("   F: %0X\n", e.PrimaryKey.Fingerprint)
	bits, _ := e.PrimaryKey.BitLength()
	fmt.Printf("   L: %v\n", bits)
}

func printPubKeyring() (err error) {
	var pubEntlist openpgp.EntityList

	if pubEntlist, err = LoadPubKeyring(); err != nil {
		return err
	}

	for i, e := range pubEntlist {
		printEntity(i, e)
		fmt.Println("   --------")
	}

	return nil
}

func printPrivKeyring() (err error) {
	var privEntlist openpgp.EntityList

	if privEntlist, err = LoadPrivKeyring(); err != nil {
		return err
	}

	for i, e := range privEntlist {
		printEntity(i, e)
		fmt.Println("   --------")
	}

	return nil
}

// GenKeyPair generates a PGP key pair and store them in the sypgp home folder
func GenKeyPair() error {
	conf := &packet.Config{RSABits: 4096, DefaultHash: crypto.SHA384}

	if err := PathsCheck(); err != nil {
		return err
	}

	fmt.Print("Enter your name (e.g., John Doe) : ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	name := scanner.Text()
	if err := scanner.Err(); err != nil {
		log.Println("Error while reading name from user: ", err)
		return err
	}

	fmt.Print("Enter your email address (e.g., john.doe@example.com) : ")
	scanner.Scan()
	email := scanner.Text()
	if err := scanner.Err(); err != nil {
		log.Println("Error while reading email from user: ", err)
		return err
	}

	fmt.Print("Enter optional comment (e.g., development keys) : ")
	scanner.Scan()
	comment := scanner.Text()
	if err := scanner.Err(); err != nil {
		log.Println("Error while reading comment from user: ", err)
		return err
	}

	fmt.Print("Generating Entity and PGP Key Pair... ")
	entity, err := openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		log.Println("Error while creating entity: ", err)
		return err
	}
	fmt.Println("Done")

	fs, err := os.OpenFile(SecretPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Could not open private keyring file for appending: ", err)
		return err
	}
	defer fs.Close()

	if err = entity.SerializePrivate(fs, nil); err != nil {
		log.Println("Error while writing private entity to keyring file: ", err)
		return err
	}

	fp, err := os.OpenFile(PublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Could not open public keyring file for appending: ", err)
		return err
	}
	defer fp.Close()

	if err = entity.Serialize(fp); err != nil {
		log.Println("Error while writing public entity to keyring file: ", err)
		return err
	}

	return nil
}

// DecryptKey decrypts a private key provided a pass phrase
// XXX: replace that with acutal cli passwd grab
func DecryptKey(k *openpgp.Entity) error {
	if k.PrivateKey.Encrypted == true {
		fmt.Print("Enter key passphrase: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		pass := scanner.Text()
		if err := scanner.Err(); err != nil {
			fmt.Println("error while reading passphrase from user", err)
			return err
		}

		if err := k.PrivateKey.Decrypt([]byte(pass)); err != nil {
			log.Println("error while decrypting key: ", err)
			return err
		}
	}
	return nil
}

// SelectKey prints a key list to user and returns the choice
func SelectKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	var index int

	printPrivKeyring()
	fmt.Print("Enter # of signing key to use : ")
	n, err := fmt.Scanf("%d", &index)
	if err != nil || n != 1 {
		log.Println("Error while reading key choice from user: ", err)
		return nil, err
	}

	if index < 0 || index > len(el)-1 {
		fmt.Println("invalid key choice")
		return nil, fmt.Errorf("invalid key choice")
	}

	return el[index], nil
}

// FetchPubkey connects to a key server and requests a specific key
func FetchPubkey(fingerprint string, sykeysAddr string) (openpgp.EntityList, error) {
	v := url.Values{}
	v.Set("op", "get")
	v.Set("options", "mr")
	v.Set("search", "0x"+fingerprint)
	u := url.URL{
		Scheme:   "http",
		Host:     sykeysAddr,
		Path:     "pks/lookup",
		RawQuery: v.Encode(),
	}
	urlStr := u.String()

	log.Println("url:", urlStr)
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Println("error in http.Get:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("no matching keys found for fingerprint")
		log.Println(err)
		return nil, err
	}

	el, err := openpgp.ReadArmoredKeyRing(resp.Body)
	if err != nil {
		log.Println("error while trying to read armored key ring:", err)
		return nil, err
	}
	if len(el) == 0 {
		err = fmt.Errorf("no keys in keyring")
		log.Println(err)
		return nil, err
	}
	if len(el) > 1 {
		err = fmt.Errorf("server returned more than one key for unique fingerprint")
		log.Println(err)
		return nil, err
	}

	return el, nil
}

// PushPubkey pushes a public key to a key server
func PushPubkey(entity *openpgp.Entity, sykeysAddr string) (err error) {
	w := bytes.NewBuffer(nil)
	wr, err := armor.Encode(w, openpgp.PublicKeyType, nil)
	if err != nil {
		log.Println("armor.Encode failed:", err)
	}

	err = entity.Serialize(wr)
	if err != nil {
		log.Println("can't serialize public key:", err)
		return err
	}
	wr.Close()

	v := url.Values{}
	v.Set("keytext", w.String())
	u := url.URL{
		Scheme:   "http",
		Host:     sykeysAddr,
		Path:     "pks/add",
		RawQuery: v.Encode(),
	}
	urlStr := u.String()

	resp, err := http.PostForm(urlStr, v)
	if err != nil {
		log.Println("error in http.PostForm():", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("Key server did not accept PGP key")
		log.Println(err)
		return err
	}

	return nil
}
