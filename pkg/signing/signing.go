package signing

import (
	"bufio"
	"bytes"
	"crypto"
	"crypto/sha512"
	"fmt"
	"github.com/singularityware/singularity/pkg/image"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
	"golang.org/x/crypto/openpgp/packet"
	"log"
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

func genKeyPair(spath string, ppath string) error {
	conf := &packet.Config{RSABits: 4096, DefaultHash: crypto.SHA384}

	fmt.Println("No Private Keys found in SYPGP store, generating RSA pair for you.")

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

	log.Print("Generating Entity and Key Pair... ")
	entity, err := openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		log.Println("Error while creating entity: ", err)
		return err
	}
	log.Println("Done")

	fs, err := os.OpenFile(spath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create private keyring file: ", err)
		return err
	}
	defer fs.Close()
	if err = entity.SerializePrivate(fs, nil); err != nil {
		log.Println("Error while writing private entity to keyring file: ", err)
		return err
	}

	fp, err := os.OpenFile(ppath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Println("Could not create public keyring file: ", err)
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
		k.PrivateKey.Decrypt([]byte("devkeys"))
	}
	return nil
}

// XXX: replace that with actual cli choice maker
func selectKey(el openpgp.EntityList) (*openpgp.Entity, error) {
	return el[0], nil
}

func SyPgpPath() string {
	return filepath.Join(os.Getenv("HOME"), ".sypgp")
}

func SyPgpPathCheck() error {
	if err := os.MkdirAll(SyPgpPath(), 0700); err != nil {
		log.Println("could not create singularity PGP directory")
		return err
	}
	return nil
}

func SifDataObjectHash(sinfo *image.Sifinfo) (*bytes.Buffer, error) {
	var msg = new(bytes.Buffer)

	part, err := image.SifGetPartition(sinfo, image.SIF_DEFAULT_GROUP)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	data, err := image.CByteRange(sinfo.Mapstart(), part.FileOff(), part.FileLen())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	sum := sha512.Sum384(data)

	fmt.Fprintf(msg, "SIFHASH:\n%x", sum)

	return msg, nil
}

func SifAddSignature(sinfo *image.Sifinfo, signature []byte) error {
	var e image.Eleminfo

	part, err := image.SifGetPartition(sinfo, image.SIF_DEFAULT_GROUP)
	if err != nil {
		log.Println(err)
		return err
	}

	e.InitSignature(signature, part)

	if err := image.SifPutDataObj(&e, sinfo); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

/*
 * Signing workflow:
 * 1) Look for cmd parameter:
 *    a. no keyid ? -> privkeys in store ? yes=offer from list, no=genkey
 *    b. keyid ? get key from store.
 * 2) read key with specified keyid
 * 3) compute data object(s) hash
 * 4) sign this hash
 * 5) store new hash in SIF
 * 6) record the KeyID used to sign into signature data object descriptor
 */
func Sign(cpath string) error {
	secretpath := filepath.Join(SyPgpPath(), "pgp-secret")
	pubpath := filepath.Join(SyPgpPath(), "pgp-public")

	if err := SyPgpPathCheck(); err != nil {
		return err
	}

	f, err := os.Open(secretpath)
	if err != nil {
		log.Println("Error trying to open secret keyring file: ", err)
		log.Println("Will try to generate a new secret keyring file...")
		err = genKeyPair(secretpath, pubpath)
		if err != nil {
			return err
		}
		f, err = os.Open(secretpath)
		if err != nil {
			log.Println("Could not reopen keyring file: ", err)
			return err
		}
	}
	defer f.Close()

	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Println("Error while trying to read key ring: ", err)
		return err
	}

	var k *openpgp.Entity
	if len(el) > 1 {
		if k, err = selectKey(el); err != nil {
			return err
		}
	} else {
		k = el[0]
	}
	decryptKey(k)

	var sinfo image.Sifinfo
	if err = image.SifLoad(cpath, &sinfo, 0); err != nil {
		log.Println(err)
		return err
	}
	defer image.SifUnload(&sinfo)

	msg, err := SifDataObjectHash(&sinfo)
	if err != nil {
		return err
	}

	var signedmsg bytes.Buffer
	plaintext, err := clearsign.Encode(&signedmsg, k.PrivateKey, nil)
	if err != nil {
		log.Printf("error from Encode: %s\n", err)
		return err
	}
	if _, err := plaintext.Write(msg.Bytes()); err != nil {
		log.Printf("error from Write: %s\n", err)
		return err
	}
	if err := plaintext.Close(); err != nil {
		log.Printf("error from Close: %s\n", err)
		return err
	}

	if err = SifAddSignature(&sinfo, signedmsg.Bytes()); err != nil {
		return err
	}

	return nil
}

func Verify(cpath string) error {
	var sinfo image.Sifinfo
	if err := image.SifLoad(cpath, &sinfo, 0); err != nil {
		log.Println(err)
		return err
	}
	defer image.SifUnload(&sinfo)

	msg, err := SifDataObjectHash(&sinfo)
	if err != nil {
		return err
	}

	sig, err := image.SifGetSignature(&sinfo)
	if err != nil {
		log.Println(err)
		return err
	}

	data, err := image.CByteRange(sinfo.Mapstart(), sig.FileOff(), sig.FileLen())
	if err != nil {
		log.Println(err)
		return err
	}

	block, _ := clearsign.Decode(data)
	if block == nil {
		log.Printf("failed to decode clearsign message\n")
		return fmt.Errorf("failed to decode clearsign message\n")
	}

	if !bytes.Equal(bytes.TrimRight(block.Plaintext, "\n"), msg.Bytes()) {
		log.Printf("Sif hash string mismatch -- don't use:\nsigned:     %s\ncalculated: %s", msg.String(), block.Plaintext)
		return fmt.Errorf("Sif hash string mismatch -- don't use")
	}

	if err := SyPgpPathCheck(); err != nil {
		return err
	}

	pubpath := filepath.Join(SyPgpPath(), "pgp-public")
	f, err := os.Open(pubpath)
	if err != nil {
		log.Println("Error trying to open public keyring file: ", err)

		/* XXX: Try to get KEY from KEYSERVER */
	}
	defer f.Close()

	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Println("Error while trying to read key ring: ", err)
		return err
	}

	var signer *openpgp.Entity
	if signer, err = openpgp.CheckDetachedSignature(el, bytes.NewBuffer(block.Bytes), block.ArmoredSignature.Body); err != nil {
		log.Printf("failed to check signature: %s", err)
		return err
	}
	fmt.Print("Authentic and signed by:\n")
	for _, i := range signer.Identities {
		fmt.Printf("\t%s\n", i.Name)
	}

	return nil
}
