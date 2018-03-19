package signing

import (
	"bytes"
	"crypto"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"log"
	"os"
)

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

func main() {
	hpath := os.Getenv("HOME")
	f, err := os.Open(hpath + "/pgp-secret")
	if err != nil {
		log.Fatal("could not open keyring file")
	}
	el, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range el {
		if e.PrivateKey.Encrypted == true {
			e.PrivateKey.Decrypt([]byte("devkeys"))
		}
		buf := bytes.NewBufferString("Allo")
		var conf packet.Config
		conf.DefaultHash = crypto.SHA384
		err = openpgp.DetachSignText(os.Stdout, e, buf, &conf)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("")
	}
}
