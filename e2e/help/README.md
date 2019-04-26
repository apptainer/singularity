# Testing `singularity help` commands

This package contains the end-to-end tests for `singularity help`. 

## Contributing new help tests

For this example, we're going to create a new test for `singularity help inspect`.

* Add the help text to the `testdata/help` directory.

```
singlularity help inspect > e2e/testdata/help/help-inspect.txt
```

* Add the help command to the `testHelp` function in `help.go`

```
var helpTests = []struct {
        cmds []string
}{
	...
	// singluarity inspect
	{[]string{"help", "inspect"}},
	...
}	
```

## Updating existing help tests

For this example, we're going to update an existing test for `singularity help inspect`.

* When a help test fails, we need to check why it failed.

```
Was the failure a result of an unintended change? If so, we open an issue.

Was the failure a result of an intended change? If so, we update the help text.
```

* Update the help text in the `testdata/help` directory.

```
singlularity help inspect > e2e/testdata/help/help-inspect.txt
```

## Running the help tests

* To verify this test, modify the `Makefile` to add both a verbose flag and a filter flag (`go test -v -r testHelp`) and than run the tests.

```
SINGULARITY_E2E=1 make -C builddir e2e-test
```
