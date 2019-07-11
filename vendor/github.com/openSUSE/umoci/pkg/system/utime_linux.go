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
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// Lutimes is a wrapper around utimensat(2), with the AT_SYMLINK_NOFOLLOW flag
// set, to allow changing the time of a symlink rather than the file it points
// to.
func Lutimes(path string, atime, mtime time.Time) error {
	times := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}

	err := unix.UtimesNanoAt(unix.AT_FDCWD, path, times, unix.AT_SYMLINK_NOFOLLOW)
	if err != nil {
		return &os.PathError{Op: "lutimes", Path: path, Err: err}
	}
	return nil
}
