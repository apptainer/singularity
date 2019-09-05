package cli

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/sylabs/singularity/pkg/sypgp"

	"gotest.tools/v3/assert"
)

const (
	testName     = "John"
	testEmail    = "john@sylabs.io"
	testComment  = "so test"
	testPassword = "1234"
)

func Test_collectInput_flags(t *testing.T) {
	nameF := pflag.Flag{Name: KeyNewPairNameFlag.Name, Changed: true}
	emailF := pflag.Flag{Name: KeyNewPairEmailFlag.Name, Changed: true}
	commentF := pflag.Flag{Name: KeyNewPairCommentFlag.Name, Changed: true}
	passwordF := pflag.Flag{Name: KeyNewPairPasswordFlag.Name, Changed: true}
	pushF := pflag.Flag{Name: KeyNewPairPushFlag.Name, Changed: true}

	c := cobra.Command{}
	c.Flags().AddFlag(&nameF)
	c.Flags().AddFlag(&emailF)
	c.Flags().AddFlag(&commentF)
	c.Flags().AddFlag(&passwordF)
	c.Flags().AddFlag(&pushF)

	keyNewPairName = testName
	keyNewPairEmail = testEmail
	keyNewPairComment = testComment
	keyNewPairPassword = testPassword
	keyNewPairPush = true

	expectedOpts := &keyNewPairOptions{
		GenKeyPairOptions: sypgp.GenKeyPairOptions{
			Name:     testName,
			Email:    testEmail,
			Comment:  testComment,
			Password: testPassword,
		},
		PushToKeyStore: true,
	}

	o, err := collectInput(&c)
	assert.NilError(t, err)
	assert.DeepEqual(t, expectedOpts, o)
}

func Test_collectInput_stdin(t *testing.T) {
	tf, err := ioutil.TempFile("", "collect-test-")
	assert.NilError(t, err)
	defer tf.Close()

	oldStdin := os.Stdin
	defer func(ostdin *os.File) {
		os.Stdin = ostdin
	}(oldStdin)
	os.Stdin = tf

	tests := []struct {
		Name    string
		Input   string
		Options *keyNewPairOptions
		Error   error
	}{
		{
			Name:  "Valid input",
			Input: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\ny", testName, testEmail, testComment, testPassword, testPassword),
			Options: &keyNewPairOptions{
				GenKeyPairOptions: sypgp.GenKeyPairOptions{
					Name:     testName,
					Email:    testEmail,
					Comment:  testComment,
					Password: testPassword,
				},
				PushToKeyStore: true},
			Error: nil,
		},
		{
			Name:  "No pass, OK",
			Input: fmt.Sprintf("%s\n%s\n%s\n%s\n%s\ny\ny", testName, testEmail, testComment, "", ""),
			Options: &keyNewPairOptions{
				GenKeyPairOptions: sypgp.GenKeyPairOptions{
					Name:     testName,
					Email:    testEmail,
					Comment:  testComment,
					Password: "",
				},
				PushToKeyStore: true},
			Error: nil,
		},
		{
			Name:    "No pass, FAIL",
			Input:   fmt.Sprintf("%s\n%s\n%s\n%s\n%s\nn\ny", testName, testEmail, testComment, "", ""),
			Options: nil,
			Error:   errors.New("empty passphrase"),
		},
	}

	c := &cobra.Command{}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			assert.NilError(t, tf.Truncate(0))
			_, err := tf.Seek(0, 0)
			assert.NilError(t, err)
			_, err = tf.WriteString(tt.Input)
			assert.NilError(t, err)
			_, err = tf.Seek(0, 0)
			assert.NilError(t, err)

			o, err := collectInput(c)
			if tt.Error != nil {
				assert.ErrorContains(t, err, tt.Error.Error())
			} else {
				assert.NilError(t, err)
			}

			assert.DeepEqual(t, tt.Options, o)
		})
	}
}
