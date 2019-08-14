package mtree

import (
	"io"
	"sort"
)

// DirectoryHierarchy is the mapped structure for an mtree directory hierarchy
// spec
type DirectoryHierarchy struct {
	Entries []Entry
}

// WriteTo simplifies the output of the resulting hierarchy spec
func (dh DirectoryHierarchy) WriteTo(w io.Writer) (n int64, err error) {
	sort.Sort(byPos(dh.Entries))
	var sum int64
	for _, e := range dh.Entries {
		str := e.String()
		i, err := io.WriteString(w, str+"\n")
		if err != nil {
			return sum, err
		}
		sum += int64(i)
	}
	return sum, nil
}

// UsedKeywords collects and returns all the keywords used in a
// a DirectoryHierarchy
func (dh DirectoryHierarchy) UsedKeywords() []Keyword {
	usedkeywords := []Keyword{}
	for _, e := range dh.Entries {
		switch e.Type {
		case FullType, RelativeType, SpecialType:
			if e.Type != SpecialType || e.Name == "/set" {
				kvs := e.Keywords
				for _, kv := range kvs {
					kw := KeyVal(kv).Keyword().Prefix()
					if !InKeywordSlice(kw, usedkeywords) {
						usedkeywords = append(usedkeywords, KeywordSynonym(string(kw)))
					}
				}
			}
		}
	}
	return usedkeywords
}
