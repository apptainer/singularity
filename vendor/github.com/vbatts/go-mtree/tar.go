package mtree

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vbatts/go-mtree/pkg/govis"
)

// Streamer creates a file hierarchy out of a tar stream
type Streamer interface {
	io.ReadCloser
	Hierarchy() (*DirectoryHierarchy, error)
}

var tarDefaultSetKeywords = []KeyVal{
	"type=file",
	"flags=none",
	"mode=0664",
}

// NewTarStreamer streams a tar archive and creates a file hierarchy based off
// of the tar metadata headers
func NewTarStreamer(r io.Reader, excludes []ExcludeFunc, keywords []Keyword) Streamer {
	pR, pW := io.Pipe()
	ts := &tarStream{
		pipeReader: pR,
		pipeWriter: pW,
		creator:    dhCreator{DH: &DirectoryHierarchy{}},
		teeReader:  io.TeeReader(r, pW),
		tarReader:  tar.NewReader(pR),
		keywords:   keywords,
		hardlinks:  map[string][]string{},
		excludes:   excludes,
	}

	go ts.readHeaders()
	return ts
}

type tarStream struct {
	root       *Entry
	hardlinks  map[string][]string
	creator    dhCreator
	pipeReader *io.PipeReader
	pipeWriter *io.PipeWriter
	teeReader  io.Reader
	tarReader  *tar.Reader
	keywords   []Keyword
	excludes   []ExcludeFunc
	err        error
}

func (ts *tarStream) readHeaders() {
	// remove "time" keyword
	notimekws := []Keyword{}
	for _, kw := range ts.keywords {
		if !InKeywordSlice(kw, notimekws) {
			if kw == "time" {
				if !InKeywordSlice("tar_time", ts.keywords) {
					notimekws = append(notimekws, "tar_time")
				}
			} else {
				notimekws = append(notimekws, kw)
			}
		}
	}
	ts.keywords = notimekws
	// We have to start with the directory we're in, and anything beyond these
	// items is determined at the time a tar is extracted.
	ts.root = &Entry{
		Name: ".",
		Type: RelativeType,
		Prev: &Entry{
			Raw:  "# .",
			Type: CommentType,
		},
		Set:      nil,
		Keywords: []KeyVal{"type=dir"},
	}
	// insert signature and metadata comments first (user, machine, tree, date)
	for _, e := range signatureEntries("<user specified tar archive>") {
		e.Pos = len(ts.creator.DH.Entries)
		ts.creator.DH.Entries = append(ts.creator.DH.Entries, e)
	}
	// insert keyword metadata next
	for _, e := range keywordEntries(ts.keywords) {
		e.Pos = len(ts.creator.DH.Entries)
		ts.creator.DH.Entries = append(ts.creator.DH.Entries, e)
	}
hdrloop:
	for {
		hdr, err := ts.tarReader.Next()
		if err != nil {
			ts.pipeReader.CloseWithError(err)
			return
		}

		for _, ex := range ts.excludes {
			if ex(hdr.Name, hdr.FileInfo()) {
				continue hdrloop
			}
		}

		// Because the content of the file may need to be read by several
		// KeywordFuncs, it needs to be an io.Seeker as well. So, just reading from
		// ts.tarReader is not enough.
		tmpFile, err := ioutil.TempFile("", "ts.payload.")
		if err != nil {
			ts.pipeReader.CloseWithError(err)
			return
		}
		// for good measure
		if err := tmpFile.Chmod(0600); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			ts.pipeReader.CloseWithError(err)
			return
		}
		if _, err := io.Copy(tmpFile, ts.tarReader); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			ts.pipeReader.CloseWithError(err)
			return
		}
		// Alright, it's either file or directory
		encodedName, err := govis.Vis(filepath.Base(hdr.Name), DefaultVisFlags)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			ts.pipeReader.CloseWithError(err)
			return
		}
		e := Entry{
			Name: encodedName,
			Type: RelativeType,
		}

		// Keep track of which files are hardlinks so we can resolve them later
		if hdr.Typeflag == tar.TypeLink {
			keyFunc := KeywordFuncs["link"]
			kvs, err := keyFunc(hdr.Name, hdr.FileInfo(), nil)
			if err != nil {
				logrus.Warn(err)
				break // XXX is breaking an okay thing to do here?
			}
			linkname, err := govis.Unvis(KeyVal(kvs[0]).Value(), DefaultVisFlags)
			if err != nil {
				logrus.Warn(err)
				break // XXX is breaking an okay thing to do here?
			}
			if _, ok := ts.hardlinks[linkname]; !ok {
				ts.hardlinks[linkname] = []string{hdr.Name}
			} else {
				ts.hardlinks[linkname] = append(ts.hardlinks[linkname], hdr.Name)
			}
		}

		// now collect keywords on the file
		for _, keyword := range ts.keywords {
			if keyFunc, ok := KeywordFuncs[keyword.Prefix()]; ok {
				// We can't extract directories on to disk, so "size" keyword
				// is irrelevant for now
				if hdr.FileInfo().IsDir() && keyword == "size" {
					continue
				}
				kvs, err := keyFunc(hdr.Name, hdr.FileInfo(), tmpFile)
				if err != nil {
					ts.setErr(err)
				}
				// for good measure, check that we actually get a value for a keyword
				if len(kvs) > 0 && kvs[0] != "" {
					e.Keywords = append(e.Keywords, kvs[0])
				}

				// don't forget to reset the reader
				if _, err := tmpFile.Seek(0, 0); err != nil {
					tmpFile.Close()
					os.Remove(tmpFile.Name())
					ts.pipeReader.CloseWithError(err)
					return
				}
			}
		}
		// collect meta-set keywords for a directory so that we can build the
		// actual sets in `flatten`
		if hdr.FileInfo().IsDir() {
			s := Entry{
				Name: "meta-set",
				Type: SpecialType,
			}
			for _, setKW := range SetKeywords {
				if keyFunc, ok := KeywordFuncs[setKW.Prefix()]; ok {
					kvs, err := keyFunc(hdr.Name, hdr.FileInfo(), tmpFile)
					if err != nil {
						ts.setErr(err)
					}
					for _, kv := range kvs {
						if kv != "" {
							s.Keywords = append(s.Keywords, kv)
						}
					}
					if _, err := tmpFile.Seek(0, 0); err != nil {
						tmpFile.Close()
						os.Remove(tmpFile.Name())
						ts.pipeReader.CloseWithError(err)
					}
				}
			}
			e.Set = &s
		}
		err = populateTree(ts.root, &e, hdr)
		if err != nil {
			ts.setErr(err)
		}
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}
}

// populateTree creates a pseudo file tree hierarchy using an Entry's Parent and
// Children fields. When examining the Entry e to insert in the tree, we
// determine if the path to that Entry exists yet. If it does, insert it in the
// appropriate position in the tree. If not, create a path up until the Entry's
// directory that it is contained in. Then, insert the Entry.
// root: the "." Entry
//    e: the Entry we are looking to insert
//  hdr: the tar header struct associated with e
func populateTree(root, e *Entry, hdr *tar.Header) error {
	if root == nil || e == nil {
		return fmt.Errorf("cannot populate or insert nil Entry's")
	} else if root.Prev == nil {
		return fmt.Errorf("root needs to be an Entry associated with a directory")
	}
	isDir := hdr.FileInfo().IsDir()
	wd := filepath.Clean(hdr.Name)
	if !isDir {
		// directory up until the actual file
		wd = filepath.Dir(wd)
		if wd == "." {
			root.Children = append([]*Entry{e}, root.Children...)
			e.Parent = root
			return nil
		}
	}
	dirNames := strings.Split(wd, "/")
	parent := root
	for _, name := range dirNames[:] {
		encoded, err := govis.Vis(name, DefaultVisFlags)
		if err != nil {
			return err
		}
		if node := parent.Descend(encoded); node == nil {
			// Entry for directory doesn't exist in tree relative to root.
			// We don't know if this directory is an actual tar header (because a
			// user could have just specified a path to a deep file), so we must
			// specify this placeholder directory as a "type=dir", and Set=nil.
			newEntry := Entry{
				Name:     encoded,
				Type:     RelativeType,
				Parent:   parent,
				Keywords: []KeyVal{"type=dir"}, // temp data
				Set:      nil,                  // temp data
			}
			pathname, err := newEntry.Path()
			if err != nil {
				return err
			}
			newEntry.Prev = &Entry{
				Type: CommentType,
				Raw:  "# " + pathname,
			}
			parent.Children = append(parent.Children, &newEntry)
			parent = &newEntry
		} else {
			// Entry for directory exists in tree, just keep going
			parent = node
		}
	}
	if !isDir {
		parent.Children = append([]*Entry{e}, parent.Children...)
		e.Parent = parent
	} else {
		// fill in the actual data from e
		parent.Keywords = e.Keywords
		parent.Set = e.Set
	}
	return nil
}

// After constructing a pseudo file hierarchy tree, we want to "flatten" this
// tree by putting the Entries into a slice with appropriate positioning.
//     root: the "head" of the sub-tree to flatten
//  creator: a dhCreator that helps with the '/set' keyword
// keywords: keywords specified by the user that should be evaluated
func flatten(root *Entry, creator *dhCreator, keywords []Keyword) {
	if root == nil || creator == nil {
		return
	}
	if root.Prev != nil {
		// root.Prev != nil implies root is a directory
		creator.DH.Entries = append(creator.DH.Entries,
			Entry{
				Type: BlankType,
				Pos:  len(creator.DH.Entries),
			})
		root.Prev.Pos = len(creator.DH.Entries)
		creator.DH.Entries = append(creator.DH.Entries, *root.Prev)

		if root.Set != nil {
			// Check if we need a new set
			consolidatedKeys := keyvalSelector(append(tarDefaultSetKeywords, root.Set.Keywords...), keywords)
			if creator.curSet == nil {
				creator.curSet = &Entry{
					Type:     SpecialType,
					Name:     "/set",
					Keywords: consolidatedKeys,
					Pos:      len(creator.DH.Entries),
				}
				creator.DH.Entries = append(creator.DH.Entries, *creator.curSet)
			} else {
				needNewSet := false
				for _, k := range root.Set.Keywords {
					if !inKeyValSlice(k, creator.curSet.Keywords) {
						needNewSet = true
						break
					}
				}
				if needNewSet {
					creator.curSet = &Entry{
						Name:     "/set",
						Type:     SpecialType,
						Pos:      len(creator.DH.Entries),
						Keywords: consolidatedKeys,
					}
					creator.DH.Entries = append(creator.DH.Entries, *creator.curSet)
				}
			}
		} else if creator.curSet != nil {
			// Getting into here implies that the Entry's set has not and
			// was not supposed to be evaluated, thus, we need to reset curSet
			creator.DH.Entries = append(creator.DH.Entries, Entry{
				Name: "/unset",
				Type: SpecialType,
				Pos:  len(creator.DH.Entries),
			})
			creator.curSet = nil
		}
	}
	root.Set = creator.curSet
	if creator.curSet != nil {
		root.Keywords = keyValDifference(root.Keywords, creator.curSet.Keywords)
	}
	root.Pos = len(creator.DH.Entries)
	creator.DH.Entries = append(creator.DH.Entries, *root)
	for _, c := range root.Children {
		flatten(c, creator, keywords)
	}
	if root.Prev != nil {
		// Show a comment when stepping out
		root.Prev.Pos = len(creator.DH.Entries)
		creator.DH.Entries = append(creator.DH.Entries, *root.Prev)
		dotEntry := Entry{
			Type: DotDotType,
			Name: "..",
			Pos:  len(creator.DH.Entries),
		}
		creator.DH.Entries = append(creator.DH.Entries, dotEntry)
	}
	return
}

// resolveHardlinks goes through an Entry tree, and finds the Entry's associated
// with hardlinks and fills them in with the actual data from the base file.
func resolveHardlinks(root *Entry, hardlinks map[string][]string, countlinks bool) {
	originals := make(map[string]*Entry)
	for base, links := range hardlinks {
		var basefile *Entry
		if seen, ok := originals[base]; !ok {
			basefile = root.Find(base)
			if basefile == nil {
				logrus.Printf("%s does not exist in this tree\n", base)
				continue
			}
			originals[base] = basefile
		} else {
			basefile = seen
		}
		for _, link := range links {
			linkfile := root.Find(link)
			if linkfile == nil {
				logrus.Printf("%s does not exist in this tree\n", link)
				continue
			}
			linkfile.Keywords = basefile.Keywords
			if countlinks {
				linkfile.Keywords = append(linkfile.Keywords, KeyVal(fmt.Sprintf("nlink=%d", len(links)+1)))
			}
		}
		if countlinks {
			basefile.Keywords = append(basefile.Keywords, KeyVal(fmt.Sprintf("nlink=%d", len(links)+1)))
		}
	}
}

// filter takes in a pointer to an Entry, and returns a slice of Entry's that
// satisfy the predicate p
func filter(root *Entry, p func(*Entry) bool) []Entry {
	if root != nil {
		var validEntrys []Entry
		if len(root.Children) > 0 || root.Prev != nil {
			for _, c := range root.Children {
				// filter the sub-directory
				if c.Prev != nil {
					validEntrys = append(validEntrys, filter(c, p)...)
				}
				if p(c) {
					if c.Prev == nil {
						validEntrys = append([]Entry{*c}, validEntrys...)
					} else {
						validEntrys = append(validEntrys, *c)
					}
				}
			}
			return validEntrys
		}
	}
	return nil
}

func (ts *tarStream) setErr(err error) {
	ts.err = err
}

func (ts *tarStream) Read(p []byte) (n int, err error) {
	return ts.teeReader.Read(p)
}

func (ts *tarStream) Close() error {
	return ts.pipeReader.Close()
}

// Hierarchy returns the DirectoryHierarchy of the archive. It flattens the
// Entry tree before returning the DirectoryHierarchy
func (ts *tarStream) Hierarchy() (*DirectoryHierarchy, error) {
	if ts.err != nil && ts.err != io.EOF {
		return nil, ts.err
	}
	if ts.root == nil {
		return nil, fmt.Errorf("root Entry not found, nothing to flatten")
	}
	resolveHardlinks(ts.root, ts.hardlinks, InKeywordSlice(Keyword("nlink"), ts.keywords))
	flatten(ts.root, &ts.creator, ts.keywords)
	return ts.creator.DH, nil
}
