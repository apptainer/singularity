package signing

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/sha512"
	"fmt"
	"github.com/singularityware/singularity/src/pkg/sif"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/clearsign"
	"golang.org/x/crypto/openpgp/packet"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

const (
	syKeysAddr = "example.org:11371"
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

func syPgpDirPath() string {
	return filepath.Join(os.Getenv("HOME"), ".sypgp")
}

func syPgpSecretPath() string {
	return filepath.Join(syPgpDirPath(), "pgp-secret")
}

func syPgpPublicPath() string {
	return filepath.Join(syPgpDirPath(), "pgp-public")
}

// Create Singularity PGP home folder, secret and public keyring files
func syPgpPathsCheck() error {
	if err := os.MkdirAll(syPgpDirPath(), 0700); err != nil {
		log.Println("could not create singularity PGP directory")
		return err
	}

	fs, err := os.OpenFile(syPgpSecretPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create private keyring file: ", err)
		return err
	}
	fs.Close()

	fp, err := os.OpenFile(syPgpPublicPath(), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create public keyring file: ", err)
		return err
	}
	fp.Close()

	return nil
}

func syPgpLoadPrivKeyring() (openpgp.EntityList, error) {
	if err := syPgpPathsCheck(); err != nil {
		return nil, err
	}

	f, err := os.Open(syPgpSecretPath())
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

func syPgpLoadPubKeyring() (openpgp.EntityList, error) {
	if err := syPgpPathsCheck(); err != nil {
		return nil, err
	}

	f, err := os.Open(syPgpPublicPath())
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

	if pubEntlist, err = syPgpLoadPubKeyring(); err != nil {
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

	if privEntlist, err = syPgpLoadPrivKeyring(); err != nil {
		return err
	}

	for i, e := range privEntlist {
		printEntity(i, e)
		fmt.Println("   --------")
	}

	return nil
}

func genKeyPair() error {
	conf := &packet.Config{RSABits: 4096, DefaultHash: crypto.SHA384}

	if err := syPgpPathsCheck(); err != nil {
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

	fs, err := os.OpenFile(syPgpSecretPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Println("Could not open private keyring file for appending: ", err)
		return err
	}
	defer fs.Close()

	if err = entity.SerializePrivate(fs, nil); err != nil {
		log.Println("Error while writing private entity to keyring file: ", err)
		return err
	}

	fp, err := os.OpenFile(syPgpPublicPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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

// XXX: replace that with acutal cli passwd grab
func decryptKey(k *openpgp.Entity) error {
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

func selectKey(el openpgp.EntityList) (*openpgp.Entity, error) {
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

func sifDataObjectHash(sinfo *sif.Sifinfo) (*bytes.Buffer, error) {
	var msg = new(bytes.Buffer)

	part, err := sif.SifGetPartition(sinfo, sif.SIF_DEFAULT_GROUP)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	data, err := sif.CByteRange(sinfo.Mapstart(), part.FileOff(), part.FileLen())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sum := sha512.Sum384(data)

	fmt.Fprintf(msg, "SIFHASH:\n%x", sum)

	return msg, nil
}

func sifAddSignature(fingerprint [20]byte, sinfo *sif.Sifinfo, signature []byte) error {
	var e sif.Eleminfo

	part, err := sif.SifGetPartition(sinfo, sif.SIF_DEFAULT_GROUP)
	if err != nil {
		log.Println(err)
		return err
	}

	e.InitSignature(fingerprint, signature, part)

	if err := sif.SifPutDataObj(&e, sinfo); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func sykeysFetchPubkey(fingerprint string, sykeysAddr string) (openpgp.EntityList, error) {
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

func sykeysPushPubkey(entity *openpgp.Entity, sykeysAddr string) (err error) {
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

// Sign takes the path of a container and generates a PGP signature block for
// its system partition. Sign uses the private keys found in the default
// location if available or helps the user by prompting with key generation
// configuration options. In its current form, Sign also pushes public material
// to a key server if enabled. This should be a separate step in the next round
// of development.
func Sign(cpath string) error {
	var el openpgp.EntityList
	var en *openpgp.Entity
	var err error

	if el, err = syPgpLoadPrivKeyring(); err != nil {
		return err
	} else if el == nil {
		fmt.Println("No Private Keys found in SYPGP store, generating RSA pair for you.")
		err = genKeyPair()
		if err != nil {
			return err
		}
		if el, err = syPgpLoadPrivKeyring(); err != nil || el == nil {
			return err
		}
		fmt.Printf("Sending PGP public key material: %0X => %s.\n", el[0].PrimaryKey.Fingerprint, syKeysAddr)
		err = sykeysPushPubkey(el[0], syKeysAddr)
		if err != nil {
			return err
		}
	}

	if len(el) > 1 {
		if en, err = selectKey(el); err != nil {
			return err
		}
	} else {
		en = el[0]
	}
	decryptKey(en)

	var sinfo sif.Sifinfo
	if err = sif.SifLoad(cpath, &sinfo, 0); err != nil {
		log.Println("error loading sif file:", cpath, err)
		return err
	}
	defer sif.SifUnload(&sinfo)

	msg, err := sifDataObjectHash(&sinfo)
	if err != nil {
		return err
	}

	var signedmsg bytes.Buffer
	plaintext, err := clearsign.Encode(&signedmsg, en.PrivateKey, nil)
	if err != nil {
		log.Printf("error from Encode: %s\n", err)
		return err
	}
	if _, err = plaintext.Write(msg.Bytes()); err != nil {
		log.Printf("error from Write: %s\n", err)
		return err
	}
	if err = plaintext.Close(); err != nil {
		log.Printf("error from Close: %s\n", err)
		return err
	}

	if err = sifAddSignature(en.PrimaryKey.Fingerprint, &sinfo, signedmsg.Bytes()); err != nil {
		return err
	}

	return nil
}

// Verify takes a container path and look for a verification block for a
// system partition. If found, the signature block is used to verify the
// partition hash against the signer's version. Verify takes care of looking
// for PGP keys in the default local store or looks it up from a key server
// if access is enabled.
func Verify(cpath string) error {
	var el openpgp.EntityList
	var sinfo sif.Sifinfo

	if err := sif.SifLoad(cpath, &sinfo, 0); err != nil {
		log.Println(err)
		return err
	}
	defer sif.SifUnload(&sinfo)

	msg, err := sifDataObjectHash(&sinfo)
	if err != nil {
		return err
	}

	sig, err := sif.SifGetSignature(&sinfo)
	if err != nil {
		log.Println(err)
		return err
	}

	data, err := sif.CByteRange(sinfo.Mapstart(), sig.FileOff(), sig.FileLen())
	if err != nil {
		log.Println(err)
		return err
	}

	block, _ := clearsign.Decode(data)
	if block == nil {
		log.Println("failed to decode clearsign message")
		return fmt.Errorf("failed to decode clearsign message")
	}

	if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), msg.Bytes()) {
		log.Printf("Sif hash string mismatch -- don't use:\nsigned:     %s\ncalculated: %s", msg.String(), block.Plaintext)
		return fmt.Errorf("Sif hash string mismatch -- don't use")
	}

	if el, err = syPgpLoadPubKeyring(); err != nil {
		return err
	}

	/* try to verify with local PGP store */
	var signer *openpgp.Entity
	if signer, err = openpgp.CheckDetachedSignature(el, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
		log.Printf("failed to check signature: %s", err)
		/* verification with local keyring failed, try to fetch from key server */
		log.Println("Contacting sykeys PGP key management services for:", sig.GetEntity())
		syel, err := sykeysFetchPubkey(sig.GetEntity(), syKeysAddr)
		if err != nil {
			return err
		}

		block, _ := clearsign.Decode(data)
		if block == nil {
			log.Println("failed to decode clearsign message")
			return fmt.Errorf("failed to decode clearsign message")
		}

		if signer, err = openpgp.CheckDetachedSignature(syel, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
			log.Printf("failed to check signature: %s", err)
			return err
		}
	}
	fmt.Print("Authentic and signed by:\n")
	for _, i := range signer.Identities {
		fmt.Printf("\t%s\n", i.Name)
	}

	return nil
}
