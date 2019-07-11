package mtree

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vbatts/go-mtree/pkg/govis"
)

type byPos []Entry

func (bp byPos) Len() int           { return len(bp) }
func (bp byPos) Less(i, j int) bool { return bp[i].Pos < bp[j].Pos }
func (bp byPos) Swap(i, j int)      { bp[i], bp[j] = bp[j], bp[i] }

// Entry is each component of content in the mtree spec file
type Entry struct {
	Parent     *Entry   // up
	Children   []*Entry // down
	Prev, Next *Entry   // left, right
	Set        *Entry   // current `/set` for additional keywords
	Pos        int      // order in the spec
	Raw        string   // file or directory name
	Name       string   // file or directory name
	Keywords   []KeyVal // TODO(vbatts) maybe a keyword typed set of values?
	Type       EntryType
}

// Descend searches thru an Entry's children to find the Entry associated with
// `filename`. Directories are stored at the end of an Entry's children so do a
// traverse backwards. If you descend to a "."
func (e Entry) Descend(filename string) *Entry {
	if filename == "." || filename == "" {
		return &e
	}
	numChildren := len(e.Children)
	for i := range e.Children {
		c := e.Children[numChildren-1-i]
		if c.Name == filename {
			return c
		}
	}
	return nil
}

// Find is a wrapper around Descend that takes in a whole string path and tries
// to find that Entry
func (e Entry) Find(filepath string) *Entry {
	resultnode := &e
	for _, path := range strings.Split(filepath, "/") {
		encoded, err := govis.Vis(path, DefaultVisFlags)
		if err != nil {
			return nil
		}
		resultnode = resultnode.Descend(encoded)
		if resultnode == nil {
			return nil
		}
	}
	return resultnode
}

// Ascend gets the parent of an Entry. Serves mainly to maintain readability
// when traversing up and down an Entry tree
func (e Entry) Ascend() *Entry {
	return e.Parent
}

// Path provides the full path of the file, despite RelativeType or FullType. It
// will be in Unvis'd form.
func (e Entry) Path() (string, error) {
	decodedName, err := govis.Unvis(e.Name, DefaultVisFlags)
	if err != nil {
		return "", err
	}
	if e.Parent == nil || e.Type == FullType {
		return filepath.Clean(decodedName), nil
	}
	parentName, err := e.Parent.Path()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(parentName, decodedName)), nil
}

// String joins a file with its associated keywords. The file name will be the
// Vis'd encoded version so that it can be parsed appropriately when Check'd.
func (e Entry) String() string {
	if e.Raw != "" {
		return e.Raw
	}
	if e.Type == BlankType {
		return ""
	}
	if e.Type == DotDotType {
		return e.Name
	}
	if e.Type == SpecialType || e.Type == FullType || inKeyValSlice("type=dir", e.Keywords) {
		return fmt.Sprintf("%s %s", e.Name, strings.Join(KeyValToString(e.Keywords), " "))
	}
	return fmt.Sprintf("    %s %s", e.Name, strings.Join(KeyValToString(e.Keywords), " "))
}

// AllKeys returns the full set of KeyVal for the given entry, based on the
// /set keys as well as the entry-local keys. Entry-local keys always take
// precedence.
func (e Entry) AllKeys() []KeyVal {
	if e.Set != nil {
		return MergeKeyValSet(e.Set.Keywords, e.Keywords)
	}
	return e.Keywords
}

// IsDir checks the type= value for this entry on whether it is a directory
func (e Entry) IsDir() bool {
	for _, kv := range e.AllKeys() {
		if kv.Keyword().Prefix() == "type" {
			return kv.Value() == "dir"
		}
	}
	return false
}

// EntryType are the formats of lines in an mtree spec file
type EntryType int

// The types of lines to be found in an mtree spec file
const (
	SignatureType EntryType = iota // first line of the file, like `#mtree v2.0`
	BlankType                      // blank lines are ignored
	CommentType                    // Lines beginning with `#` are ignored
	SpecialType                    // line that has `/` prefix issue a "special" command (currently only /set and /unset)
	RelativeType                   // if the first white-space delimited word does not have a '/' in it. Options/keywords are applied.
	DotDotType                     // .. - A relative path step. keywords/options are ignored
	FullType                       // if the first word on the line has a `/` after the first character, it interpretted as a file pathname with options
)

// String returns the name of the EntryType
func (et EntryType) String() string {
	return typeNames[et]
}

var typeNames = map[EntryType]string{
	SignatureType: "SignatureType",
	BlankType:     "BlankType",
	CommentType:   "CommentType",
	SpecialType:   "SpecialType",
	RelativeType:  "RelativeType",
	DotDotType:    "DotDotType",
	FullType:      "FullType",
}
