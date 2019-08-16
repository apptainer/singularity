/*
 * umoci: Umoci Modifies Open Containers' Images
 * Copyright (C) 2017, 2018 SUSE LLC.
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

package casext

import (
	"bytes"
	"encoding/json"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// PutBlobJSON adds a new JSON blob to the image (marshalled from the given
// interface). This is equivalent to calling PutBlob() with a JSON payload
// as the reader. Note that due to intricacies in the Go JSON
// implementation, we cannot guarantee that two calls to PutBlobJSON() will
// return the same digest.
//
// TODO: Use a proper JSON serialisation library, which actually guarantees
//       consistent output. Go's JSON library doesn't even attempt to sort
//       map[...]... objects (which have their iteration order randomised in
//       Go).
func (e Engine) PutBlobJSON(ctx context.Context, data interface{}) (digest.Digest, int64, error) {
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(data); err != nil {
		return "", -1, errors.Wrap(err, "encode JSON")
	}
	return e.PutBlob(ctx, &buffer)
}
