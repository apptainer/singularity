// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package auth

import (
	"testing"

	"github.com/sylabs/singularity/internal/pkg/test"
)

const (
	testToken     = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.TCYt5XsITJX1CxPCT8yAV-TVkIEq_PbChOMqsLfRoPsnsgw5WEuts01mq-pQy7UJiN5mgRxD-WUcX16dUEMGlv50aqzpqh4Qktb3rk-BuQy72IFLOqV0G_zS245-kronKb78cPN25DGlcTwLtjPAYuNzVBAh4vGHSrQyHUdBBPM"
	testTokenPath = "test_data/test_token"
)

func Test_ReadToken(t *testing.T) {

	test.DropPrivilege(t)
	defer test.ResetPrivilege(t)

	result, w := ReadToken("/no/such/file")
	if result != "" {
		t.Errorf("readToken from invalid file must give empty string")
	}

	result, w = ReadToken("test_data/test_token_toosmall")
	if w != WarningTokenTooShort {
		t.Errorf("readToken from file with bad (too small) token must give empty string")
	}

	result, _ = ReadToken(testTokenPath)
	if result != testToken {
		t.Errorf("readToken from valid file must match expected result")
	}
}
