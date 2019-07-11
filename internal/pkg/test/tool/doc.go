// Copyright (c) 2019, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package tool

/*
Package tool provides all related test helpers/tools which can be used by
by unit/e2e/integration tests.

All helpers functions here should take as first argument *testing.T, if a function
doesn't require or use *testing.T, this function should go into another package
and not in test or tool package.

Any helper functions using another package should be placed here.

All helpers can potentially influence test execution by calling t.Fatalf,
t.Errorf, t.Skipf ... and/or log any actions useful for tests execution.
*/
