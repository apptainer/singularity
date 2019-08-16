/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2016, 2017, 2018 SUSE LLC.
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

package layer

import (
	"archive/tar"

	"golang.org/x/sys/unix"
)

func updateHeader(hdr *tar.Header, s unix.Stat_t) {
	// Currently the Go stdlib doesn't fill in the major/minor numbers of
	// devices, so we have to do it manually.
	if s.Mode&unix.S_IFBLK == unix.S_IFBLK || s.Mode&unix.S_IFCHR == unix.S_IFCHR {
		hdr.Devmajor = int64(unix.Major(uint64(s.Rdev)))
		hdr.Devminor = int64(unix.Minor(uint64(s.Rdev)))
	}
}
