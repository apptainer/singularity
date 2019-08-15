/*
 * govis: unicode aware vis(3) encoding implementation
 * Copyright (C) 2017 SUSE LLC.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package govis

// VisFlag manipulates how the characters are encoded/decoded
type VisFlag uint

// vis() has a variety of flags when deciding what encodings to use. While
// mtree only uses one set of flags, implementing them all is necessary in
// order to have compatibility with BSD's vis() and unvis() commands.
const (
	VisOctal     VisFlag = (1 << iota)     // VIS_OCTAL: Use octal \ddd format.
	VisCStyle                              // VIS_CSTYLE: Use \[nrft0..] where appropriate.
	VisSpace                               // VIS_SP: Also encode space.
	VisTab                                 // VIS_TAB: Also encode tab.
	VisNewline                             // VIS_NL: Also encode newline.
	VisSafe                                // VIS_SAFE: Encode unsafe characters.
	VisNoSlash                             // VIS_NOSLASH: Inhibit printing '\'.
	VisHTTPStyle                           // VIS_HTTPSTYLE: HTTP-style escape %xx.
	VisGlob                                // VIS_GLOB: Encode glob(3) magics.
	visMask      VisFlag = (1 << iota) - 1 // Mask of all flags.

	VisWhite VisFlag = (VisSpace | VisTab | VisNewline)
)
