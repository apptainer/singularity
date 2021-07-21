# End-to-End Testing

This package contains the end-to-end tests for `singularity`.

## Contributing

For this example, we're going to use a topic of `env` or
`environment variable tests`.

- Add your topic as a runtime-hook in `suite.go`.

```go
// RunE2ETests by functionality
t.Run("BUILD", imgbuild.RunE2ETests)
t.Run("ACTIONS", actions.RunE2ETests)
t.Run("ENV", env.RunE2ETests)
```

- Create a directory for your topic.

```sh
mkdir -p e2e/env
```

- Create a source file to include your topic's tests.

```sh
touch e2e/env/env.go
```

- Optionally create a source file to include helpers for your topic's test.

```sh
touch e2e/env/env_utils.go
```

- Add a package declaration to your topic's test file that matches what you put
  in `suite.go`

```go
package env
```

- Add a variable to store the testing settings in your topic's test file.

```go
import (
        "github.com/kelseyhightower/envconfig"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv
```

- Add a entry-point to your topic's test file that matches what you put in
  `suite.go`

```go
//RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}
}
```

- Create a test in your topic's test file as you normally would in `go`.

```go
func TestEnv(t *Testing.T) {
	...
}
```

- Run your test from your entry-point function using a `go` sub-test.

```go
//RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
        if err != nil {
        	t.Fatal(err.Error())
        }
        
	// Add tests
	t.Run("TestEnv", TestEnv)
}
```

- Example of what your topic's test file might look like:

```go
package env 

import (
	"github.com/kelseyhightower/envconfig"
)

type testingEnv struct {
	// base env for running tests
	CmdPath     string `split_words:"true"`
	TestDir     string `split_words:"true"`
	RunDisabled bool   `default:"false"`
}

var testenv testingEnv

func TestEnv(t *testing.T) {
	...
}

//RunE2ETests is the main func to trigger the test suite
func RunE2ETests(t *testing.T) {
	err := envconfig.Process("E2E", &testenv)
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Run("TestEnv", TestEnv)
}
```

## Running

Test your topic using the `e2e` target in the `Makefile`. To avoid skipping
these tests (default), make sure you set the environment variable
`SINGULARITY_E2E` to `1`.

```sh
SINGULARITY_E2E=1 make -C builddir e2e-test
```

- Verify that your test was run by modifying the `Makefile` to add a verbose
  flag (`go test -v`) and re-running the previous `make` step.
