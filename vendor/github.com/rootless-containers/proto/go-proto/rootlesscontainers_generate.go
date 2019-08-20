/*
 * rootlesscontainers-proto: persistent rootless filesystem emulation
 * Copyright (C) 2018 Rootless Containers Authors
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

package rootlesscontainers

// Generate everything for our protobuf.
//go:generate protoc --go_out=import_path=rootlesscontainers:. rootlesscontainers.proto

// Keyname is the official xattr key used to store rootlesscontainers.proto
// blobs, and is the only key we will treat in this special way.
const Keyname = "user.rootlesscontainers"

// NoopID is the uint32 that represents the "noop" id for uid/gid values. It is
// equal to uint32(-1) but since we cannot write that in Go we have to
// explicitly write the wrapped value.
var NoopID uint32 = 0xFFFFFFFF

// IsDefault returns whether the given Resource is the default. If a Resource
// is equal to the default Resource then it is not necesary to include it on
// the filesystem.
func IsDefault(r Resource) bool {
	return r.Uid == NoopID && r.Gid == NoopID
}
