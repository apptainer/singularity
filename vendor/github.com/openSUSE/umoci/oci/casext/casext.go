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

// Package casext provides extensions to the standard cas.Engine interface,
// allowing for generic functionality to be used on top of any implementation
// of cas.Engine.
package casext

import "github.com/openSUSE/umoci/oci/cas"

// TODO: Convert this to an interface and make Engine private.

// Engine is a wrapper around cas.Engine that provides additional, generic
// extensions to the transport-dependent cas.Engine implementation.
type Engine struct {
	cas.Engine
}

// NewEngine returns a new Engine which acts as a wrapper around the given
// cas.Engine and provides additional, generic extensions to the
// transport-dependent cas.Engine implementation.
func NewEngine(engine cas.Engine) Engine {
	return Engine{Engine: engine}
}
