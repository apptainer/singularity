package exec

import (
	"testing"
)

func TestExecExpectCode(t *testing.T) {
	testCases := map[string]struct {
		cmd          string
		args         []string
		expectedCode int
	}{
		"true": {
			cmd:          "true",
			expectedCode: 0,
		},
		"false": {
			cmd:          "false",
			expectedCode: 1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			Command(tc.cmd, tc.args...).ExecExpectCode(t, tc.expectedCode)
		})
	}
}
