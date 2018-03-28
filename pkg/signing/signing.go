package signing

import (
	"bufio"
	//	"bytes"
	"crypto"
	"fmt"
	"github.com/singularityware/singularity/pkg/image"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"log"
	"os"
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
	in := bufio.NewReader(os.Stdin)
	name, err := in.ReadString('\n')
	if err != nil {
		log.Println("Error while reading name from user: ", err)
		return err
	}

	fmt.Print("Enter your email address (e.g., john.doe@example.com) : ")
	email, err := in.ReadString('\n')
	if err != nil {
		log.Println("Error while reading email from user: ", err)
		return err
	}

	fmt.Print("Enter optional comment (e.g., development keys) : ")
	comment, err := in.ReadString('\n')
	if err != nil {
		log.Println("Error while reading comment from user: ", err)
		return err
	}

	entity, err := openpgp.NewEntity(name, comment, email, conf)
	if err != nil {
		log.Println("Error while creating entity: ", err)
		return err
	}

	fs, err := os.Create(spath)
	if err != nil {
		log.Println("Could not create private keyring file: ", err)
		return err
	}
	defer fs.Close()
	if err = entity.SerializePrivate(fs, nil); err != nil {
		log.Println("Error while writing private entity to keyring file: ", err)
		return err
	}

	fp, err := os.Create(ppath)
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

func Sign(message []byte) error {
	secretpath := os.Getenv("HOME") + "/.sypgp/pgp-secret"
	pubpath := os.Getenv("HOME") + "/.sypgp/pgp-public"
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

	containerPath := "/home/yanik/sdev/containers/img.sif"
	var sinfo image.Sifinfo
	if ret := image.SifLoad(containerPath, &sinfo, 0); ret != nil {
		log.Println(err)
		return err
	}
	image.SifPrintHeader(&sinfo)

	if err = image.SifUnload(&sinfo); err != nil {
		return err
	}

	/*
		buf := bytes.NewBuffer(message)
		var conf packet.Config
		conf.DefaultHash = crypto.SHA384
		err = openpgp.ArmoredDetachSignText(os.Stdout, k, buf, &conf)
		if err != nil {
			log.Fatal("Error while creating signature block: ", err)
		}
	*/
	return nil
}

func Verify() (bool, error) {
	return true, nil
}
