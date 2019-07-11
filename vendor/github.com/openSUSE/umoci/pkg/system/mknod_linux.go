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

package system

import (
	"archive/tar"

	"golang.org/x/sys/unix"
)

// Tarmode takes a Typeflag (from a tar.Header for example) and returns the
// corresponding os.Filemode bit. Unknown typeflags are treated like regular
// files.
func Tarmode(typeflag byte) uint32 {
	switch typeflag {
	case tar.TypeSymlink:
		return unix.S_IFLNK
	case tar.TypeChar:
		return unix.S_IFCHR
	case tar.TypeBlock:
		return unix.S_IFBLK
	case tar.TypeFifo:
		return unix.S_IFIFO
	case tar.TypeDir:
		return unix.S_IFDIR
	}
	return 0
}
