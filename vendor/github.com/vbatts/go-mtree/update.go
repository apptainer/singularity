package mtree

import (
	"container/heap"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
)

// DefaultUpdateKeywords is the default set of keywords that can take updates to the files on disk
var DefaultUpdateKeywords = []Keyword{
	"uid",
	"gid",
	"mode",
	"xattr",
	"link",
	"time",
}

// Update attempts to set the attributes of root directory path, given the values of `keywords` in dh DirectoryHierarchy.
func Update(root string, dh *DirectoryHierarchy, keywords []Keyword, fs FsEval) ([]InodeDelta, error) {
	creator := dhCreator{DH: dh}
	curDir, err := os.Getwd()
	if err == nil {
		defer os.Chdir(curDir)
	}

	if err := os.Chdir(root); err != nil {
		return nil, err
	}
	sort.Sort(byPos(creator.DH.Entries))

	// This is for deferring the update of mtimes of directories, to unwind them
	// in a most specific path first
	h := &pathUpdateHeap{}
	heap.Init(h)

	results := []InodeDelta{}
	for i, e := range creator.DH.Entries {
		switch e.Type {
		case SpecialType:
			if e.Name == "/set" {
				creator.curSet = &creator.DH.Entries[i]
			} else if e.Name == "/unset" {
				creator.curSet = nil
			}
			logrus.Debugf("%#v", e)
			continue
		case RelativeType, FullType:
			e.Set = creator.curSet
			pathname, err := e.Path()
			if err != nil {
				return nil, err
			}

			// filter the keywords to update on the file, from the keywords available for this entry:
			var kvToUpdate []KeyVal
			kvToUpdate = keyvalSelector(e.AllKeys(), keywords)
			logrus.Debugf("kvToUpdate(%q): %#v", pathname, kvToUpdate)

			for _, kv := range kvToUpdate {
				if !InKeywordSlice(kv.Keyword().Prefix(), keywordPrefixes(keywords)) {
					continue
				}
				logrus.Debugf("finding function for %q (%q)", kv.Keyword(), kv.Keyword().Prefix())
				ukFunc, ok := UpdateKeywordFuncs[kv.Keyword().Prefix()]
				if !ok {
					logrus.Debugf("no UpdateKeywordFunc for %s; skipping", kv.Keyword())
					continue
				}

				// TODO check for the type=dir of the entry as well
				if kv.Keyword().Prefix() == "time" && e.IsDir() {
					heap.Push(h, pathUpdate{
						Path: pathname,
						E:    e,
						KV:   kv,
						Func: ukFunc,
					})

					continue
				}

				if _, err := ukFunc(pathname, kv); err != nil {
					results = append(results, InodeDelta{
						diff: ErrorDifference,
						path: pathname,
						old:  e,
						keys: []KeyDelta{
							{
								diff: ErrorDifference,
								name: kv.Keyword(),
								err:  err,
							},
						}})
				}
				// XXX really would be great to have a Check() or Compare() right here,
				// to compare each entry as it is encountered, rather than just running
				// Check() on this path after the whole update is finished.
			}
		}
	}

	for h.Len() > 0 {
		pu := heap.Pop(h).(pathUpdate)
		if _, err := pu.Func(pu.Path, pu.KV); err != nil {
			results = append(results, InodeDelta{
				diff: ErrorDifference,
				path: pu.Path,
				old:  pu.E,
				keys: []KeyDelta{
					{
						diff: ErrorDifference,
						name: pu.KV.Keyword(),
						err:  err,
					},
				}})
		}
	}
	return results, nil
}

type pathUpdateHeap []pathUpdate

func (h pathUpdateHeap) Len() int      { return len(h) }
func (h pathUpdateHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// This may end up looking backwards, but for container/heap, Less evaluates
// the negative priority. So when popping members of the array, it will be
// sorted by least. For this use-case, we want the most-qualified-name popped
// first (the longest path name), such that "." is the last entry popped.
func (h pathUpdateHeap) Less(i, j int) bool {
	return len(h[i].Path) > len(h[j].Path)
}

func (h *pathUpdateHeap) Push(x interface{}) {
	*h = append(*h, x.(pathUpdate))
}

func (h *pathUpdateHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

type pathUpdate struct {
	Path string
	E    Entry
	KV   KeyVal
	Func UpdateKeywordFunc
}
