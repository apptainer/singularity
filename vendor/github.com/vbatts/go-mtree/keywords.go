package mtree

import (
	"fmt"
	"strings"

	"github.com/vbatts/go-mtree/pkg/govis"
)

// DefaultVisFlags is the set of Vis flags used when encoding filenames and
// other similar entries.
const DefaultVisFlags govis.VisFlag = govis.VisWhite | govis.VisOctal | govis.VisGlob

// Keyword is the string name of a keyword, with some convenience functions for
// determining whether it is a default or bsd standard keyword.
// It first portion before the "="
type Keyword string

// Prefix is the portion of the keyword before a first "." (if present).
//
// Primarly for the xattr use-case, where the keyword `xattr.security.selinux` would have a Suffix of `security.selinux`.
func (k Keyword) Prefix() Keyword {
	if strings.Contains(string(k), ".") {
		return Keyword(strings.SplitN(string(k), ".", 2)[0])
	}
	return k
}

// Suffix is the portion of the keyword after a first ".".
// This is an option feature.
//
// Primarly for the xattr use-case, where the keyword `xattr.security.selinux` would have a Suffix of `security.selinux`.
func (k Keyword) Suffix() string {
	if strings.Contains(string(k), ".") {
		return strings.SplitN(string(k), ".", 2)[1]
	}
	return string(k)
}

// Default returns whether this keyword is in the default set of keywords
func (k Keyword) Default() bool {
	return InKeywordSlice(k, DefaultKeywords)
}

// Bsd returns whether this keyword is in the upstream FreeBSD mtree(8)
func (k Keyword) Bsd() bool {
	return InKeywordSlice(k, BsdKeywords)
}

// Synonym returns the canonical name for this keyword. This is provides the
// same functionality as KeywordSynonym()
func (k Keyword) Synonym() Keyword {
	return KeywordSynonym(string(k))
}

// InKeywordSlice checks for the presence of `a` in `list`
func InKeywordSlice(a Keyword, list []Keyword) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
func inKeyValSlice(a KeyVal, list []KeyVal) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// ToKeywords makes a list of Keyword from a list of string
func ToKeywords(list []string) []Keyword {
	ret := make([]Keyword, len(list))
	for i := range list {
		ret[i] = Keyword(list[i])
	}
	return ret
}

// FromKeywords makes a list of string from a list of Keyword
func FromKeywords(list []Keyword) []string {
	ret := make([]string, len(list))
	for i := range list {
		ret[i] = string(list[i])
	}
	return ret
}

// KeyValToString constructs a list of string from the list of KeyVal
func KeyValToString(list []KeyVal) []string {
	ret := make([]string, len(list))
	for i := range list {
		ret[i] = string(list[i])
	}
	return ret
}

// StringToKeyVals constructs a list of KeyVal from the list of strings, like "keyword=value"
func StringToKeyVals(list []string) []KeyVal {
	ret := make([]KeyVal, len(list))
	for i := range list {
		ret[i] = KeyVal(list[i])
	}
	return ret
}

// KeyVal is a "keyword=value"
type KeyVal string

// Keyword is the mapping to the available keywords
func (kv KeyVal) Keyword() Keyword {
	if !strings.Contains(string(kv), "=") {
		return Keyword("")
	}
	return Keyword(strings.SplitN(strings.TrimSpace(string(kv)), "=", 2)[0])
}

// Value is the data/value portion of "keyword=value"
func (kv KeyVal) Value() string {
	if !strings.Contains(string(kv), "=") {
		return ""
	}
	return strings.SplitN(strings.TrimSpace(string(kv)), "=", 2)[1]
}

// NewValue returns a new KeyVal with the newval
func (kv KeyVal) NewValue(newval string) KeyVal {
	return KeyVal(fmt.Sprintf("%s=%s", kv.Keyword(), newval))
}

// Equal returns whether two KeyVal are equivalent. This takes
// care of certain odd cases such as tar_mtime, and should be used over
// using == comparisons directly unless you really know what you're
// doing.
func (kv KeyVal) Equal(b KeyVal) bool {
	// TODO: Implement handling of tar_mtime.
	return kv.Keyword() == b.Keyword() && kv.Value() == b.Value()
}

func keywordPrefixes(kvset []Keyword) []Keyword {
	kvs := []Keyword{}
	for _, kv := range kvset {
		kvs = append(kvs, kv.Prefix())
	}
	return kvs
}

// keyvalSelector takes an array of KeyVal ("keyword=value") and filters out
// that only the set of keywords
func keyvalSelector(keyval []KeyVal, keyset []Keyword) []KeyVal {
	retList := []KeyVal{}
	for _, kv := range keyval {
		if InKeywordSlice(kv.Keyword().Prefix(), keywordPrefixes(keyset)) {
			retList = append(retList, kv)
		}
	}
	return retList
}

func keyValDifference(this, that []KeyVal) []KeyVal {
	if len(this) == 0 {
		return that
	}
	diff := []KeyVal{}
	for _, kv := range this {
		if !inKeyValSlice(kv, that) {
			diff = append(diff, kv)
		}
	}
	return diff
}
func keyValCopy(set []KeyVal) []KeyVal {
	ret := make([]KeyVal, len(set))
	for i := range set {
		ret[i] = set[i]
	}
	return ret
}

// Has the "keyword" present in the list of KeyVal, and returns the
// corresponding KeyVal, else an empty string.
func Has(keyvals []KeyVal, keyword string) []KeyVal {
	return HasKeyword(keyvals, Keyword(keyword))
}

// HasKeyword the "keyword" present in the list of KeyVal, and returns the
// corresponding KeyVal, else an empty string.
// This match is done on the Prefix of the keyword only.
func HasKeyword(keyvals []KeyVal, keyword Keyword) []KeyVal {
	kvs := []KeyVal{}
	for i := range keyvals {
		if keyvals[i].Keyword().Prefix() == keyword.Prefix() {
			kvs = append(kvs, keyvals[i])
		}
	}
	return kvs
}

// MergeSet takes the current setKeyVals, and then applies the entryKeyVals
// such that the entry's values win. The union is returned.
func MergeSet(setKeyVals, entryKeyVals []string) []KeyVal {
	retList := StringToKeyVals(setKeyVals)
	eKVs := StringToKeyVals(entryKeyVals)
	return MergeKeyValSet(retList, eKVs)
}

// MergeKeyValSet does a merge of the two sets of KeyVal, and the KeyVal of
// entryKeyVals win when there is a duplicate Keyword.
func MergeKeyValSet(setKeyVals, entryKeyVals []KeyVal) []KeyVal {
	retList := keyValCopy(setKeyVals)
	seenKeywords := []Keyword{}
	for i := range retList {
		word := retList[i].Keyword()
		for _, kv := range HasKeyword(entryKeyVals, word) {
			// match on the keyword prefix and suffix here
			if kv.Keyword() == word {
				retList[i] = kv
			}
		}
		seenKeywords = append(seenKeywords, word)
	}
	for i := range entryKeyVals {
		if !InKeywordSlice(entryKeyVals[i].Keyword(), seenKeywords) {
			retList = append(retList, entryKeyVals[i])
		}
	}
	return retList
}

var (
	// DefaultKeywords has the several default keyword producers (uid, gid,
	// mode, nlink, type, size, mtime)
	DefaultKeywords = []Keyword{
		"size",
		"type",
		"uid",
		"gid",
		"mode",
		"link",
		"nlink",
		"time",
	}

	// DefaultTarKeywords has keywords that should be used when creating a manifest from
	// an archive. Currently, evaluating the # of hardlinks has not been implemented yet
	DefaultTarKeywords = []Keyword{
		"size",
		"type",
		"uid",
		"gid",
		"mode",
		"link",
		"tar_time",
	}

	// BsdKeywords is the set of keywords that is only in the upstream FreeBSD mtree
	BsdKeywords = []Keyword{
		"cksum",
		"flags", // this one is really mostly BSD specific ...
		"ignore",
		"gid",
		"gname",
		"link",
		"md5",
		"md5digest",
		"mode",
		"nlink",
		"nochange",
		"optional",
		"ripemd160digest",
		"rmd160",
		"rmd160digest",
		"sha1",
		"sha1digest",
		"sha256",
		"sha256digest",
		"sha384",
		"sha384digest",
		"sha512",
		"sha512digest",
		"size",
		"tags",
		"time",
		"type",
		"uid",
		"uname",
	}

	// SetKeywords is the default set of keywords calculated for a `/set` SpecialType
	SetKeywords = []Keyword{
		"uid",
		"gid",
	}
)

// KeywordSynonym returns the canonical name for keywords that have synonyms,
// and just returns the name provided if there is no synonym. In this way it
// ought to be safe to wrap any keyword name.
func KeywordSynonym(name string) Keyword {
	var retname string
	switch name {
	case "md5":
		retname = "md5digest"
	case "rmd160":
		retname = "ripemd160digest"
	case "rmd160digest":
		retname = "ripemd160digest"
	case "sha1":
		retname = "sha1digest"
	case "sha256":
		retname = "sha256digest"
	case "sha384":
		retname = "sha384digest"
	case "sha512":
		retname = "sha512digest"
	case "sha512256":
		retname = "sha512256digest"
	case "xattrs":
		retname = "xattr"
	default:
		retname = name
	}
	return Keyword(retname)
}
