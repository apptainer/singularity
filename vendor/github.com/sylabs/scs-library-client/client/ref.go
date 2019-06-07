// Copyright (c) 2018, Sylabs Inc. All rights reserved.
// This software is licensed under a 3-clause BSD license. Please consult the
// LICENSE.md file distributed with the sources of this project regarding your
// rights to use or distribute this software.

package client

import (
	"errors"
	"net/url"
	"strings"
)

// Scheme is the required scheme for Library URIs.
const Scheme = "library"

var (
	// ErrRefSchemeNotValid represents a ref with an invalid scheme.
	ErrRefSchemeNotValid = errors.New("library: ref scheme not valid")
	// ErrRefUserNotPermitted represents a ref with an invalid user.
	ErrRefUserNotPermitted = errors.New("library: user not permitted in ref")
	// ErrRefQueryNotPermitted represents a ref with an invalid query.
	ErrRefQueryNotPermitted = errors.New("library: query not permitted in ref")
	// ErrRefFragmentNotPermitted represents a ref with an invalid fragment.
	ErrRefFragmentNotPermitted = errors.New("library: fragment not permitted in ref")
	// ErrRefPathNotValid represents a ref with an invalid path.
	ErrRefPathNotValid = errors.New("library: ref path not valid")
	// ErrRefTagsNotValid represents a ref with invalid tags.
	ErrRefTagsNotValid = errors.New("library: ref tags not valid")
)

// A Ref represents a parsed Library URI.
//
// The general form represented is:
//
//	scheme:[//host][/]path[:tags]
//
// The host contains both the hostname and port, if present. These values can be accessed using
// the Hostname and Port methods.
//
// Examples of valid URIs:
//
//  library:path:tags
//  library:/path:tags
//  library:///path:tags
//  library://host/path:tags
//  library://host:port/path:tags
//
// The tags component is a comma-separated list of one or more tags.
type Ref struct {
	Host string   // host or host:port
	Path string   // project or entity/project
	Tags []string // list of tags
}

// parseTags takes raw tags and returns a slice of tags.
func parseTags(rawTags string) (tags []string, err error) {
	if len(rawTags) == 0 {
		return nil, ErrRefTagsNotValid
	}

	return strings.Split(rawTags, ","), nil
}

// parsePath takes the URI path and parses the path and tags.
func parsePath(rawPath string) (path string, tags []string, err error) {
	if len(rawPath) == 0 {
		return "", nil, ErrRefPathNotValid
	}

	// The path is separated from the tags (if present) by a single colon.
	parts := strings.Split(rawPath, ":")
	if len(parts) > 2 {
		return "", nil, ErrRefPathNotValid
	}

	// TODO: not sure we should modify the path here...
	// Name can optionally start with a leading "/".
	path = parts[0]
	if len(strings.TrimPrefix(path, "/")) == 0 {
		return "", nil, ErrRefPathNotValid
	}

	if len(parts) > 1 {
		tags, err = parseTags(parts[1])
		if err != nil {
			return "", nil, err
		}
	} else {
		tags = nil
	}
	return path, tags, nil
}

// Parse parses a raw Library reference.
func Parse(rawRef string) (r *Ref, err error) {
	u, err := url.Parse(rawRef)
	if err != nil {
		return nil, err
	}
	if u.Scheme != Scheme {
		return nil, ErrRefSchemeNotValid
	}
	if u.User != nil {
		return nil, ErrRefUserNotPermitted
	}
	if u.RawQuery != "" {
		return nil, ErrRefQueryNotPermitted
	}
	if u.Fragment != "" {
		return nil, ErrRefFragmentNotPermitted
	}

	rawPath := u.Path
	if u.Opaque != "" {
		rawPath = u.Opaque
	}

	path, tags, err := parsePath(rawPath)
	if err != nil {
		return nil, err
	}

	r = &Ref{
		Host: u.Host,
		Path: path,
		Tags: tags,
	}
	return r, nil
}

// String reassembles the ref into a valid URI string. The general form of the result is one of:
//
//	scheme:path[:tags]
//	scheme://host/path[:tags]
//
// If path does not start with a /, String uses the first form; otherwise it uses the second form.
// In the second form, if u.Host is empty, host is omitted.
func (r *Ref) String() string {
	u := url.URL{
		Scheme: Scheme,
		Host:   r.Host,
	}

	rawPath := r.Path
	if len(r.Tags) > 0 {
		rawPath += ":" + strings.Join(r.Tags, ",")
	}

	if strings.HasPrefix(rawPath, "/") {
		u.Path = rawPath
	} else {
		u.Opaque = rawPath
	}

	return u.String()
}

// Hostname returns r.Host, without any port number.
//
// If Host is an IPv6 literal with a port number, Hostname returns the IPv6 literal without the
// square brackets. IPv6 literals may include a zone identifier.
func (r *Ref) Hostname() string {
	colon := strings.IndexByte(r.Host, ':')
	if colon == -1 {
		return r.Host
	}
	if i := strings.IndexByte(r.Host, ']'); i != -1 {
		return strings.TrimPrefix(r.Host[:i], "[")
	}
	return r.Host[:colon]
}

// Port returns the port part of u.Host, without the leading colon. If u.Host doesn't contain a
// port, Port returns an empty string.
func (r *Ref) Port() string {
	colon := strings.IndexByte(r.Host, ':')
	if colon == -1 {
		return ""
	}
	if i := strings.Index(r.Host, "]:"); i != -1 {
		return r.Host[i+len("]:"):]
	}
	if strings.Contains(r.Host, "]") {
		return ""
	}
	return r.Host[colon+len(":"):]
}
