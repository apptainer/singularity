/*
 * lxd: daemon based on liblxd with a REST API
 * Copyright (C) 2015-2017 LXD Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// This code was copied from https://github.com/lxc/lxd, which is available
// under the the Apache 2.0 license (as noted above). The version of this code
// comes from the tag lxd-2.21 at /shared/util.go.

package shared

import (
	"bufio"
	"fmt"
	"os"
)

// RunningInUserNS returns whether the current process is (likely) inside a
// user namespace. This has a possible false-negative (where it will return
// false while inside a user namespace if it was intentionally configured to be
// confusing to programs like this).
func RunningInUserNS() bool {
	file, err := os.Open("/proc/self/uid_map")
	if err != nil {
		return false
	}
	defer file.Close()

	buf := bufio.NewReader(file)
	l, _, err := buf.ReadLine()
	if err != nil {
		return false
	}

	line := string(l)
	var a, b, c int64
	// #nosec G104
	fmt.Sscanf(line, "%d %d %d", &a, &b, &c)
	if a == 0 && b == 0 && c == 4294967295 {
		return false
	}
	return true
}
