package mtree

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// XXX: Do we need a Difference interface to make it so people can do var x
// Difference = <something>? The main problem is that keys and inodes need to
// have different interfaces, so it's just a pain.

// DifferenceType represents the type of a discrepancy encountered for
// an object. This is also used to represent discrepancies between keys
// for objects.
type DifferenceType string

const (
	// Missing represents a discrepancy where the object is present in
	// the @old manifest but is not present in the @new manifest.
	Missing DifferenceType = "missing"

	// Extra represents a discrepancy where the object is not present in
	// the @old manifest but is present in the @new manifest.
	Extra DifferenceType = "extra"

	// Modified represents a discrepancy where the object is present in
	// both the @old and @new manifests, but one or more of the keys
	// have different values (or have not been set in one of the
	// manifests).
	Modified DifferenceType = "modified"

	// ErrorDifference represents an attempted update to the values of
	// a keyword that failed
	ErrorDifference DifferenceType = "errored"
)

// These functions return *type from the parameter. It's just shorthand, to
// ensure that we don't accidentally expose pointers to the caller that are
// internal data.
func ePtr(e Entry) *Entry   { return &e }
func sPtr(s string) *string { return &s }

// InodeDelta Represents a discrepancy in a filesystem object between two
// DirectoryHierarchy manifests. Discrepancies are caused by entries only
// present in one manifest [Missing, Extra], keys only present in one of the
// manifests [Modified] or a difference between the keys of the same object in
// both manifests [Modified].
type InodeDelta struct {
	diff DifferenceType
	path string
	new  Entry
	old  Entry
	keys []KeyDelta
}

// Type returns the type of discrepancy encountered when comparing this inode
// between the two DirectoryHierarchy manifests.
func (i InodeDelta) Type() DifferenceType {
	return i.diff
}

// Path returns the path to the inode (relative to the root of the
// DirectoryHierarchy manifests).
func (i InodeDelta) Path() string {
	return i.path
}

// Diff returns the set of key discrepancies between the two manifests for the
// specific inode. If the DifferenceType of the inode is not Modified, then
// Diff returns nil.
func (i InodeDelta) Diff() []KeyDelta {
	return i.keys
}

// Old returns the value of the inode Entry in the "old" DirectoryHierarchy (as
// determined by the ordering of parameters to Compare).
func (i InodeDelta) Old() *Entry {
	if i.diff == Modified || i.diff == Missing {
		return ePtr(i.old)
	}
	return nil
}

// New returns the value of the inode Entry in the "new" DirectoryHierarchy (as
// determined by the ordering of parameters to Compare).
func (i InodeDelta) New() *Entry {
	if i.diff == Modified || i.diff == Extra {
		return ePtr(i.new)
	}
	return nil
}

// MarshalJSON creates a JSON-encoded version of InodeDelta.
func (i InodeDelta) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type DifferenceType `json:"type"`
		Path string         `json:"path"`
		Keys []KeyDelta     `json:"keys"`
	}{
		Type: i.diff,
		Path: i.path,
		Keys: i.keys,
	})
}

// String returns a "pretty" formatting for InodeDelta.
func (i InodeDelta) String() string {
	switch i.diff {
	case Modified:
		// Output the first failure.
		f := i.keys[0]
		return fmt.Sprintf("%q: keyword %q: expected %s; got %s", i.path, f.name, f.old, f.new)
	case Extra:
		return fmt.Sprintf("%q: unexpected path", i.path)
	case Missing:
		return fmt.Sprintf("%q: missing path", i.path)
	default:
		panic("programming error")
	}
}

// KeyDelta Represents a discrepancy in a key for a particular filesystem
// object between two DirectoryHierarchy manifests. Discrepancies are caused by
// keys only present in one manifest [Missing, Extra] or a difference between
// the keys of the same object in both manifests [Modified]. A set of these is
// returned with InodeDelta.Diff().
type KeyDelta struct {
	diff DifferenceType
	name Keyword
	old  string
	new  string
	err  error // used for update delta results
}

// Type returns the type of discrepancy encountered when comparing this key
// between the two DirectoryHierarchy manifests' relevant inode entry.
func (k KeyDelta) Type() DifferenceType {
	return k.diff
}

// Name returns the name (the key) of the KeyDeltaVal entry in the
// DirectoryHierarchy.
func (k KeyDelta) Name() Keyword {
	return k.name
}

// Old returns the value of the KeyDeltaVal entry in the "old" DirectoryHierarchy
// (as determined by the ordering of parameters to Compare). Returns nil if
// there was no entry in the "old" DirectoryHierarchy.
func (k KeyDelta) Old() *string {
	if k.diff == Modified || k.diff == Missing {
		return sPtr(k.old)
	}
	return nil
}

// New returns the value of the KeyDeltaVal entry in the "new" DirectoryHierarchy
// (as determined by the ordering of parameters to Compare). Returns nil if
// there was no entry in the "old" DirectoryHierarchy.
func (k KeyDelta) New() *string {
	if k.diff == Modified || k.diff == Extra {
		return sPtr(k.old)
	}
	return nil
}

// MarshalJSON creates a JSON-encoded version of KeyDelta.
func (k KeyDelta) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type DifferenceType `json:"type"`
		Name Keyword        `json:"name"`
		Old  string         `json:"old"`
		New  string         `json:"new"`
	}{
		Type: k.diff,
		Name: k.name,
		Old:  k.old,
		New:  k.new,
	})
}

// Like Compare, but for single inode entries only. Used to compute the
// cached version of inode.keys.
func compareEntry(oldEntry, newEntry Entry) ([]KeyDelta, error) {
	// Represents the new and old states for an entry's keys.
	type stateT struct {
		Old *KeyVal
		New *KeyVal
	}

	diffs := map[Keyword]*stateT{}
	oldKeys := oldEntry.AllKeys()
	newKeys := newEntry.AllKeys()

	// Fill the map with the old keys first.
	for _, kv := range oldKeys {
		key := kv.Keyword()
		// only add this diff if the new keys has this keyword
		if key != "tar_time" && key != "time" && key.Prefix() != "xattr" && len(HasKeyword(newKeys, key)) == 0 {
			continue
		}

		// Cannot take &kv because it's the iterator.
		copy := new(KeyVal)
		*copy = kv

		_, ok := diffs[key]
		if !ok {
			diffs[key] = new(stateT)
		}
		diffs[key].Old = copy
	}

	// Then fill the new keys.
	for _, kv := range newKeys {
		key := kv.Keyword()
		// only add this diff if the old keys has this keyword
		if key != "tar_time" && key != "time" && key.Prefix() != "xattr" && len(HasKeyword(oldKeys, key)) == 0 {
			continue
		}

		// Cannot take &kv because it's the iterator.
		copy := new(KeyVal)
		*copy = kv

		_, ok := diffs[key]
		if !ok {
			diffs[key] = new(stateT)
		}
		diffs[key].New = copy
	}

	// We need a full list of the keys so we can deal with different keyvalue
	// orderings.
	var kws []Keyword
	for kw := range diffs {
		kws = append(kws, kw)
	}

	// If both tar_time and time were specified in the set of keys, we have to
	// mess with the diffs. This is an unfortunate side-effect of tar archives.
	// TODO(cyphar): This really should be abstracted inside keywords.go
	if InKeywordSlice("tar_time", kws) && InKeywordSlice("time", kws) {
		// Delete "time".
		timeStateT := diffs["time"]
		delete(diffs, "time")

		// Make a new tar_time.
		if diffs["tar_time"].Old == nil {
			time, err := strconv.ParseFloat(timeStateT.Old.Value(), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse old time: %s", err)
			}

			newTime := new(KeyVal)
			*newTime = KeyVal(fmt.Sprintf("tar_time=%d.000000000", int64(time)))

			diffs["tar_time"].Old = newTime
		} else if diffs["tar_time"].New == nil {
			time, err := strconv.ParseFloat(timeStateT.New.Value(), 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse new time: %s", err)
			}

			newTime := new(KeyVal)
			*newTime = KeyVal(fmt.Sprintf("tar_time=%d.000000000", int64(time)))

			diffs["tar_time"].New = newTime
		} else {
			return nil, fmt.Errorf("time and tar_time set in the same manifest")
		}
	}

	// Are there any differences?
	var results []KeyDelta
	for name, diff := range diffs {
		// Invalid
		if diff.Old == nil && diff.New == nil {
			return nil, fmt.Errorf("invalid state: both old and new are nil: key=%s", name)
		}

		switch {
		// Missing
		case diff.New == nil:
			results = append(results, KeyDelta{
				diff: Missing,
				name: name,
				old:  diff.Old.Value(),
			})

		// Extra
		case diff.Old == nil:
			results = append(results, KeyDelta{
				diff: Extra,
				name: name,
				new:  diff.New.Value(),
			})

		// Modified
		default:
			if !diff.Old.Equal(*diff.New) {
				results = append(results, KeyDelta{
					diff: Modified,
					name: name,
					old:  diff.Old.Value(),
					new:  diff.New.Value(),
				})
			}
		}
	}

	return results, nil
}

// Compare compares two directory hierarchy manifests, and returns the
// list of discrepancies between the two. All of the entries in the
// manifest are considered, with differences being generated for
// RelativeType and FullType entries. Differences in structure (such as
// the way /set and /unset are written) are not considered to be
// discrepancies. The list of differences are all filesystem objects.
//
// keys controls which keys will be compared, but if keys is nil then all
// possible keys will be compared between the two manifests (allowing for
// missing entries and the like). A missing or extra key is treated as a
// Modified type.
//
// If oldDh or newDh are empty, we assume they are a hierarchy that is
// completely empty. This is purely for helping callers create synthetic
// InodeDeltas.
//
// NB: The order of the parameters matters (old, new) because Extra and
//     Missing are considered as different discrepancy types.
func Compare(oldDh, newDh *DirectoryHierarchy, keys []Keyword) ([]InodeDelta, error) {
	// Represents the new and old states for an entry.
	type stateT struct {
		Old *Entry
		New *Entry
	}

	// To deal with different orderings of the entries, use a path-keyed
	// map to make sure we don't start comparing unrelated entries.
	diffs := map[string]*stateT{}

	// First, iterate over the old hierarchy. If nil, pretend it's empty.
	if oldDh != nil {
		for _, e := range oldDh.Entries {
			if e.Type == RelativeType || e.Type == FullType {
				path, err := e.Path()
				if err != nil {
					return nil, err
				}

				// Cannot take &kv because it's the iterator.
				cEntry := new(Entry)
				*cEntry = e

				_, ok := diffs[path]
				if !ok {
					diffs[path] = &stateT{}
				}
				diffs[path].Old = cEntry
			}
		}
	}

	// Then, iterate over the new hierarchy. If nil, pretend it's empty.
	if newDh != nil {
		for _, e := range newDh.Entries {
			if e.Type == RelativeType || e.Type == FullType {
				path, err := e.Path()
				if err != nil {
					return nil, err
				}

				// Cannot take &kv because it's the iterator.
				cEntry := new(Entry)
				*cEntry = e

				_, ok := diffs[path]
				if !ok {
					diffs[path] = &stateT{}
				}
				diffs[path].New = cEntry
			}
		}
	}

	// Now we compute the diff.
	var results []InodeDelta
	for path, diff := range diffs {
		// Invalid
		if diff.Old == nil && diff.New == nil {
			return nil, fmt.Errorf("invalid state: both old and new are nil: path=%s", path)
		}

		switch {
		// Missing
		case diff.New == nil:
			results = append(results, InodeDelta{
				diff: Missing,
				path: path,
				old:  *diff.Old,
			})

		// Extra
		case diff.Old == nil:
			results = append(results, InodeDelta{
				diff: Extra,
				path: path,
				new:  *diff.New,
			})

		// Modified
		default:
			changed, err := compareEntry(*diff.Old, *diff.New)
			if err != nil {
				return nil, fmt.Errorf("comparison failed %s: %s", path, err)
			}

			// Now remove "changed" entries that don't match the keys.
			if keys != nil {
				var filterChanged []KeyDelta
				for _, keyDiff := range changed {
					if InKeywordSlice(keyDiff.name.Prefix(), keys) {
						filterChanged = append(filterChanged, keyDiff)
					}
				}
				changed = filterChanged
			}

			// Check if there were any actual changes.
			if len(changed) > 0 {
				results = append(results, InodeDelta{
					diff: Modified,
					path: path,
					old:  *diff.Old,
					new:  *diff.New,
					keys: changed,
				})
			}
		}
	}

	return results, nil
}
