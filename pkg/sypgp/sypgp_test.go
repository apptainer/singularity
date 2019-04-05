// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package sypgp

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	useragent "github.com/sylabs/singularity/pkg/util/user-agent"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
)

const (
	testName    = "Test Name"
	testComment = "blah"
	testEmail   = "test@test.com"
)

var (
	testEntity *openpgp.Entity
)

type mockPKSLookup struct {
	code int
	el   openpgp.EntityList
}

func (ms *mockPKSLookup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(ms.code)
	if ms.code == http.StatusOK {
		w.Header().Set("Content-Type", "application/pgp-keys")

		wr, err := armor.Encode(w, openpgp.PublicKeyType, nil)
		if err != nil {
			log.Fatalf("failed to get encoder: %v", err)
		}
		defer wr.Close()

		for _, e := range ms.el {
			if err = e.Serialize(wr); err != nil {
				log.Fatalf("failed to serialize entity: %v", err)
			}
		}
	}
}

func TestSearchPubkey(t *testing.T) {
	ms := &mockPKSLookup{}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	tests := []struct {
		name      string
		code      int
		el        openpgp.EntityList
		search    string
		uri       string
		authToken string
		wantErr   bool
	}{
		{"Success", http.StatusOK, openpgp.EntityList{testEntity}, "search", srv.URL, "", false},
		{"SuccessToken", http.StatusOK, openpgp.EntityList{testEntity}, "search", srv.URL, "token", false},
		{"BadURL", http.StatusOK, openpgp.EntityList{testEntity}, "search", ":", "", true},
		{"TerribleURL", http.StatusOK, openpgp.EntityList{testEntity}, "search", "terrible:", "", true},
		{"NotFound", http.StatusNotFound, nil, "search", srv.URL, "", true},
		{"Unauthorized", http.StatusUnauthorized, nil, "search", srv.URL, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code
			ms.el = tt.el

			if err := SearchPubkey(tt.search, tt.uri, tt.authToken); (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestFetchPubkey(t *testing.T) {
	ms := &mockPKSLookup{}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	fp := hex.EncodeToString(testEntity.PrimaryKey.Fingerprint[:])

	tests := []struct {
		name        string
		code        int
		el          openpgp.EntityList
		fingerprint string
		uri         string
		authToken   string
		wantErr     bool
	}{
		{"Success", http.StatusOK, openpgp.EntityList{testEntity}, fp, srv.URL, "", false},
		{"SuccessToken", http.StatusOK, openpgp.EntityList{testEntity}, fp, srv.URL, "token", false},
		{"NoKeys", http.StatusOK, openpgp.EntityList{}, fp, srv.URL, "token", true},
		{"TwoKeys", http.StatusOK, openpgp.EntityList{testEntity, testEntity}, fp, srv.URL, "token", true},
		{"BadURL", http.StatusOK, openpgp.EntityList{testEntity}, fp, ":", "", true},
		{"TerribleURL", http.StatusOK, openpgp.EntityList{testEntity}, fp, "terrible:", "", true},
		{"NotFound", http.StatusNotFound, nil, fp, srv.URL, "", true},
		{"Unauthorized", http.StatusUnauthorized, nil, fp, srv.URL, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code
			ms.el = tt.el

			el, err := FetchPubkey(tt.fingerprint, tt.uri, tt.authToken, false)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
				return
			}

			if !tt.wantErr {
				if len(el) != 1 {
					t.Fatalf("unexpected number of entities returned: %v", len(el))
				}
				for i := range tt.el {
					if fp := el[i].PrimaryKey.Fingerprint; fp != tt.el[i].PrimaryKey.Fingerprint {
						t.Errorf("fingerprint mismatch: %v / %v", fp, tt.el[i].PrimaryKey.Fingerprint)
					}
				}
			}
		})
	}
}

type mockPKSAdd struct {
	t       *testing.T
	keyText string
	code    int
}

func (m *mockPKSAdd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.code != http.StatusOK {
		w.WriteHeader(m.code)
		return
	}

	if got, want := r.Header.Get("Content-Type"), "application/x-www-form-urlencoded"; got != want {
		m.t.Errorf("got content type %v, want %v", got, want)
	}

	if err := r.ParseForm(); err != nil {
		m.t.Fatalf("failed to parse form: %v", err)
	}
	if got, want := r.Form.Get("keytext"), m.keyText; got != want {
		m.t.Errorf("got key text %v, want %v", got, want)
	}
}

func TestPushPubkey(t *testing.T) {
	keyText, err := serializeEntity(testEntity, openpgp.PublicKeyType)
	if err != nil {
		t.Fatalf("failed to serialize entity: %v", err)
	}

	ms := &mockPKSAdd{
		t:       t,
		keyText: keyText,
	}
	srv := httptest.NewServer(ms)
	defer srv.Close()

	tests := []struct {
		name      string
		uri       string
		authToken string
		code      int
		wantErr   bool
	}{
		{"Success", srv.URL, "", http.StatusOK, false},
		{"SuccessToken", srv.URL, "token", http.StatusOK, false},
		{"BadURL", ":", "", http.StatusOK, true},
		{"TerribleURL", "terrible:", "", http.StatusOK, true},
		{"Unauthorized", srv.URL, "", http.StatusUnauthorized, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms.code = tt.code

			if err := PushPubkey(testEntity, tt.uri, tt.authToken); (err != nil) != tt.wantErr {
				t.Fatalf("got err %v, want error %v", err, tt.wantErr)
			}
		})
	}
}

func TestMain(m *testing.M) {
	useragent.InitValue("singularity", "3.0.0-alpha.1-303-gaed8d30-dirty")

	e, err := openpgp.NewEntity(testName, testComment, testEmail, nil)
	if err != nil {
		log.Fatalf("failed to create entity: %v", err)
	}
	testEntity = e

	os.Exit(m.Run())
}

// AskQuestion() is also tested by indirect calls but this explicit test
// just ensure that everything is fine
func TestAskQuestion(t *testing.T) {
	testStr := "test test test\ntest2 test2\n\ntest3"
	testBytes := []byte(testStr)

	// We create a temporary file that will act as stdin
	testFile, myerr := ioutil.TempFile("", "inputTest")
	if myerr != nil {
		log.Fatal("cannot create temporary file", myerr)
	}
	defer testFile.Close()
	defer os.Remove(testFile.Name())

	// Write the data that AskQuestion() will later on read
	_, fileErr := testFile.Write(testBytes)
	if fileErr != nil {
		t.Fatal("cannot write to temporary file")
	}
	// Reposition to the beginning to ensure there is something to read
	_, fileErr = testFile.Seek(0, os.SEEK_SET)
	if fileErr != nil {
		t.Fatal("cannot go to the begining of temporary file")
	}

	// Redirect stdin
	savedStdin := os.Stdin
	defer func() {
		os.Stdin = savedStdin
	}()
	os.Stdin = testFile

	// Actual test, getting the first entry
	output, questionErr := AskQuestion("Question test: ")
	if questionErr != nil {
		t.Fatal("AskQuestion() failed", questionErr)
	}
	fmt.Println(output)
	// We make sure to NOT get the '\n'
	firstAnswer := testStr[:strings.Index(testStr, "\n")]
	restAnswer := testStr[len(firstAnswer)+1:]
	if output != firstAnswer {
		t.Fatal("AskQuestion() returned", output, "instead of", firstAnswer)
	}

	// Test with the second entry
	output, questionErr = AskQuestion("Question test 2: ")
	if questionErr != nil {
		t.Fatal("AskQuestion() failed", questionErr)
	}
	fmt.Println(output)
	secondAnswer := restAnswer[:strings.Index(restAnswer, "\n")]
	if output != secondAnswer {
		t.Fatal("AskQuestion() returned", output, "instead of", secondAnswer)
	}

	// Test with the third entry (which is empty)
	output, questionErr = AskQuestion("Question test 3: ")
	if questionErr != nil {
		t.Fatal("AskQuestion() failed", questionErr)
	}
	fmt.Println(output)
	if output != "" {
		t.Fatal("AskQuestion() returned", output, "instead of being empty")
	}

	// Test with the final entry
	output, questionErr = AskQuestion("Question test 4: ")
	if questionErr != nil {
		t.Fatal("AskQuestion() failed", questionErr)
	}
	fmt.Println(output)
	finalAnswer := restAnswer[len(secondAnswer)+2:] // We have to account for two \n
	if output != finalAnswer {
		t.Fatal("AskQuestion() returned", output, "instead of", finalAnswer)
	}
}

func TestAskQuestionNoEcho(t *testing.T) {
	testStr := "test test test\ntest2 test2\n\ntest3"
	testBytes := []byte(testStr)

	// We create a temporary file that will act as stdin
	testFile, myerr := ioutil.TempFile("", "inputTest")
	if myerr != nil {
		log.Fatal("cannot create temporary file", myerr)
	}
	defer testFile.Close()
	defer os.Remove(testFile.Name())

	// Write the data that AskQuestionNoEcho() will later on read
	_, fileErr := testFile.Write(testBytes)
	if fileErr != nil {
		t.Fatal("cannot write to temporary file")
	}
	// Reposition to the beginning to ensure there is something to read
	_, fileErr = testFile.Seek(0, 0)
	if fileErr != nil {
		t.Fatal("cannot go to the begining of temporary file")
	}

	// Redirect stdin
	savedStdin := os.Stdin
	defer func() {
		os.Stdin = savedStdin
	}()
	os.Stdin = testFile

	// Test AskQuestionNoEcho()
	output, questionErr := AskQuestionNoEcho("Test question")
	if questionErr != nil {
		t.Fatal("cannot get result", questionErr)
	}
	fmt.Println(output)
	// We make sure to NOT get the '\n'
	firstAnswer := testStr[:strings.Index(testStr, "\n")]
	restAnswer := testStr[len(firstAnswer)+1:]
	if output != firstAnswer {
		t.Fatal("AskQuestionNoEcho() returned", output, "instead of", firstAnswer)
	}

	// Test with the second entry
	output, questionErr = AskQuestionNoEcho("Question test 2: ")
	if questionErr != nil {
		t.Fatal("AskQuestionNoEcho() failed", questionErr)
	}
	fmt.Println(output)
	secondAnswer := restAnswer[:strings.Index(restAnswer, "\n")]
	if output != secondAnswer {
		t.Fatal("AskQuestionNoEcho() returned", output, "instead of", secondAnswer)
	}

	// Test with the third entry (which is empty)
	output, questionErr = AskQuestionNoEcho("Question test 3: ")
	if questionErr != nil {
		t.Fatal("AskQuestionNoEcho() failed", questionErr)
	}
	fmt.Println(output)
	if output != "" {
		t.Fatal("AskQuestionNoEcho() returned", output, "instead of being empty")
	}

	// Test with the final entry
	output, questionErr = AskQuestionNoEcho("Question test 4: ")
	if questionErr != nil {
		t.Fatal("AskQuestionNoEcho() failed", questionErr)
	}
	fmt.Println(output)
	finalAnswer := restAnswer[len(secondAnswer)+2:] // We have to account for two \n
	if output != finalAnswer {
		t.Fatal("AskQuestion() returned", output, "instead of", finalAnswer)
	}
}

func TestDirPath(t *testing.T) {
	homePath := DirPath()
	if homePath == "" {
		t.Fatal("cannot retrieve home path")
	}
}

func TesSecretPath(t *testing.T) {
	secretPath := SecretPath()
	if secretPath == "" {
		t.Fatal("cannot retrieve secret path")
	}
}

func TestPublicPath(t *testing.T) {
	publicPath := PublicPath()
	if publicPath == "" {
		t.Fatal("cannot retrieve public path")
	}
	fmt.Println("Path:", publicPath)
}

func TestPathsCheck(t *testing.T) {
	myerr := PathsCheck()
	if myerr != nil {
		t.Fatal("cannot check paths")
	}
}

func TestLoadPrivKey(t *testing.T) {
	_, err := LoadPrivKeyring()
	if err != nil {
		t.Fatal("cannot load private keyring")
	}
}

func TestLoadKeyringFromFile(t *testing.T) {
	dummyFile := "notExistingDir/notExistingFile"
	//validFile := ""

	// First we run a invalid test
	_, err := LoadKeyringFromFile(dummyFile)
	if err == nil {
		t.Fatal("successfully loaded a keyring from a dummy file")
	}

	// Then we run valid tests

}

func TestGenKeyPair(t *testing.T) {
	myToken := "MyToken"
	myURI := "MyURI"

	// Prepare all the answers that GenKeyPair is expecting
	// Note that when asking for a passphrase, the code assumes
	// a terminal and therefore, it is difficult to test (not currently
	// covered).
	testStr := "A tester\ntest@my.info\n\nfakepassphrase\nfakepassphrase\nY\n"
	testBytes := []byte(testStr)

	// We create a temporary file that will act as stdin
	testFile, myerr := ioutil.TempFile("", "inputTest")
	if myerr != nil {
		log.Fatal("cannot create temporary file", myerr)
	}
	defer testFile.Close()
	defer os.Remove(testFile.Name())
	fmt.Println("Temp file created:", testFile.Name())

	// Write the data that AskQuestion() will later on read
	_, fileErr := testFile.Write(testBytes)
	if fileErr != nil {
		t.Fatal("cannot write to temporary file")
	}
	// Reposition to the beginning to ensure there is something to read
	_, fileErr = testFile.Seek(0, 0)
	if fileErr != nil {
		t.Fatal("cannot go to the begining of temporary file", fileErr)
	}

	// Redirect stdin
	savedStdin := os.Stdin
	defer func() {
		os.Stdin = savedStdin
	}()
	os.Stdin = testFile

	_, err := GenKeyPair(myURI, myToken)
	if err == nil {
		t.Fatal("a KeyPair was created from invalid data", err)
	}
}

func TestLoadPubKeyring(t *testing.T) {
	_, err := LoadPubKeyring()
	if err != nil {
		t.Fatal("cannot load public keyring", err)
	}
}

func TestPrintEntity(t *testing.T) {
	// Test an invalid case
	PrintEntity(0, nil)
}

func TestPrintPubKeyring(t *testing.T) {
	err := PrintPubKeyring()
	if err != nil {
		t.Fatal("cannot print public keyring", err)
	}
}

func TestPrintPrivKeyring(t *testing.T) {
	err := PrintPrivKeyring()
	if err != nil {
		t.Fatal("cannot print private keyring", err)
	}
}

func TestStorePrivKey(t *testing.T) {
	// Valid case but should return right away
	err := StorePrivKey(nil)
	if err == nil {
		t.Fatal("test succeeded while expected to fail")
	}
}

func TestStorePubKey(t *testing.T) {
	// Valid case but should return right away
	err := StorePubKey(nil)
	if err == nil {
		t.Fatal("test succeeded while expected to fail")
	}
}

func TestCompareLocalPubKey(t *testing.T) {
	// Valid case but should return right away
	cmp := CompareKeyEntity(nil, "")
	if cmp == true {
		t.Fatal("comparison of different keys returned true")
	}
}

func TestCheckLocalPubKey(t *testing.T) {
	// Valid case but should return right away
	cmp, err := CheckLocalPubKey("")
	if err != nil {
		t.Fatal("checking local public key failed", err)
	}

	if cmp == true {
		t.Fatal("checking an empty key passed while it should fail")
	}
}

func TestRemovePubKey(t *testing.T) {
	// Valid case but should return right away
	err := RemovePubKey("")
	if err == nil {
		t.Fatal("test succeeded while expected to fail")
	}
}

func TestDecryptKey(t *testing.T) {
	// Valid case but should return right away
	err := DecryptKey(nil)
	if err == nil {
		t.Fatal("test succeeded while expected to fail")
	}
}

func TestEncryptKey(t *testing.T) {
	// Valid case but should return right away
	err := EncryptKey(nil, "")
	if err == nil {
		t.Fatal("test succeeded while expected to fail")
	}
}

// GetPassphrase() testing is also covered by other tests but this explicitly test
// testing with automatic feeding of stdin
func TestGetPassphrase(t *testing.T) {
	// Setup the test by redirecting stdin
	testStr := "mypassphrase\nmypassphrase\n"
	testBytes := []byte(testStr)

	// We create a temporary file that will act as stdin
	testFile, myerr := ioutil.TempFile("", "inputTest")
	if myerr != nil {
		log.Fatal("cannot create temporary file", myerr)
	}
	defer testFile.Close()
	defer os.Remove(testFile.Name())
	fmt.Println("Temp file created:", testFile.Name())

	// Write the data that AskQuestion() will later on read
	_, fileErr := testFile.Write(testBytes)
	if fileErr != nil {
		t.Fatal("cannot write to temporary file")
	}
	// Reposition to the beginning to ensure there is something to read
	_, fileErr = testFile.Seek(0, 0)
	if fileErr != nil {
		t.Fatal("cannot go to the begining of temporary file")
	}

	// Redirect stdin
	savedStdin := os.Stdin
	defer func() {
		os.Stdin = savedStdin
	}()
	os.Stdin = testFile

	pass, err := GetPassphrase(1)
	if err != nil || pass != "mypassphrase" {
		t.Fatal("cannot handle passphrase", err)
	}
}
