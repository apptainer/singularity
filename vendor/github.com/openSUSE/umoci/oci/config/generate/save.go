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

package generate

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// fakeBuffer implements the io.Writer interface but just counts the number of
// bytes "written" to it.
type fakeBuffer struct {
	n int64
}

// Write just counts the number of bytes requested to be written.
func (fb *fakeBuffer) Write(p []byte) (int, error) {
	size := len(p)
	fb.n += int64(size)
	return size, nil
}

// WriteTo outputs a JSON-marshalled version of the current state of the
// generator. It is not guaranteed that the generator will produce the same
// output given the same state, so it's recommended to only call this function
// once. The JSON is not pretty-printed.
func (g *Generator) WriteTo(w io.Writer) (n int64, err error) {
	// We need to return the number of bytes written, which json.NewEncoder
	// won't give us. So we have to cheat a little to get the answer.
	var fb fakeBuffer
	w = io.MultiWriter(w, &fb)

	if err := json.NewEncoder(w).Encode(g.image); err != nil {
		return fb.n, errors.Wrap(err, "encode image")
	}

	return fb.n, nil
}
