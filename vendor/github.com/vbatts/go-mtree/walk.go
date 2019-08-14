package mtree

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vbatts/go-mtree/pkg/govis"
)

// ExcludeFunc is the type of function called on each path walked to determine
// whether to be excluded from the assembled DirectoryHierarchy. If the func
// returns true, then the path is not included in the spec.
type ExcludeFunc func(path string, info os.FileInfo) bool

// ExcludeNonDirectories is an ExcludeFunc for excluding all paths that are not directories
var ExcludeNonDirectories = func(path string, info os.FileInfo) bool {
	return !info.IsDir()
}

var defaultSetKeyVals = []KeyVal{"type=file", "nlink=1", "flags=none", "mode=0664"}

// Walk from root directory and assemble the DirectoryHierarchy
// * `excludes` provided are used to skip paths
// * `keywords` are the set to collect from the walked paths. The recommended default list is DefaultKeywords.
// * `fsEval` is the interface to use in evaluating files. If `nil`, then DefaultFsEval is used.
func Walk(root string, excludes []ExcludeFunc, keywords []Keyword, fsEval FsEval) (*DirectoryHierarchy, error) {
	if fsEval == nil {
		fsEval = DefaultFsEval{}
	}
	creator := dhCreator{DH: &DirectoryHierarchy{}, fs: fsEval}
	// insert signature and metadata comments first (user, machine, tree, date)
	for _, e := range signatureEntries(root) {
		e.Pos = len(creator.DH.Entries)
		creator.DH.Entries = append(creator.DH.Entries, e)
	}
	// insert keyword metadata next
	for _, e := range keywordEntries(keywords) {
		e.Pos = len(creator.DH.Entries)
		creator.DH.Entries = append(creator.DH.Entries, e)
	}
	// walk the directory and add entries
	err := startWalk(&creator, root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		for _, ex := range excludes {
			if ex(path, info) {
				return nil
			}
		}

		entryPathName := filepath.Base(path)
		if info.IsDir() {
			creator.DH.Entries = append(creator.DH.Entries, Entry{
				Type: BlankType,
				Pos:  len(creator.DH.Entries),
			})

			// Insert a comment of the full path of the directory's name
			if creator.curDir != nil {
				dirname, err := creator.curDir.Path()
				if err != nil {
					return err
				}
				creator.DH.Entries = append(creator.DH.Entries, Entry{
					Pos:  len(creator.DH.Entries),
					Raw:  "# " + filepath.Join(dirname, entryPathName),
					Type: CommentType,
				})
			} else {
				entryPathName = "."
				creator.DH.Entries = append(creator.DH.Entries, Entry{
					Pos:  len(creator.DH.Entries),
					Raw:  "# .",
					Type: CommentType,
				})
			}

			// set the initial /set keywords
			if creator.curSet == nil {
				e := Entry{
					Name:     "/set",
					Type:     SpecialType,
					Pos:      len(creator.DH.Entries),
					Keywords: keyvalSelector(defaultSetKeyVals, keywords),
				}
				for _, keyword := range SetKeywords {
					err := func() error {
						var r io.Reader
						if info.Mode().IsRegular() {
							fh, err := creator.fs.Open(path)
							if err != nil {
								return err
							}
							defer fh.Close()
							r = fh
						}
						keyFunc, ok := KeywordFuncs[keyword.Prefix()]
						if !ok {
							return fmt.Errorf("Unknown keyword %q for file %q", keyword.Prefix(), path)
						}
						kvs, err := creator.fs.KeywordFunc(keyFunc)(path, info, r)
						if err != nil {
							return err
						}
						for _, kv := range kvs {
							if kv != "" {
								e.Keywords = append(e.Keywords, kv)
							}
						}
						return nil
					}()
					if err != nil {
						return err
					}
				}
				creator.curSet = &e
				creator.DH.Entries = append(creator.DH.Entries, e)
			} else if creator.curSet != nil {
				// check the attributes of the /set keywords and re-set if changed
				klist := []KeyVal{}
				for _, keyword := range SetKeywords {
					err := func() error {
						var r io.Reader
						if info.Mode().IsRegular() {
							fh, err := creator.fs.Open(path)
							if err != nil {
								return err
							}
							defer fh.Close()
							r = fh
						}
						keyFunc, ok := KeywordFuncs[keyword.Prefix()]
						if !ok {
							return fmt.Errorf("Unknown keyword %q for file %q", keyword.Prefix(), path)
						}
						kvs, err := creator.fs.KeywordFunc(keyFunc)(path, info, r)
						if err != nil {
							return err
						}
						for _, kv := range kvs {
							if kv != "" {
								klist = append(klist, kv)
							}
						}
						return nil
					}()
					if err != nil {
						return err
					}
				}

				needNewSet := false
				for _, k := range klist {
					if !inKeyValSlice(k, creator.curSet.Keywords) {
						needNewSet = true
					}
				}
				if needNewSet {
					e := Entry{
						Name:     "/set",
						Type:     SpecialType,
						Pos:      len(creator.DH.Entries),
						Keywords: keyvalSelector(append(defaultSetKeyVals, klist...), keywords),
					}
					creator.curSet = &e
					creator.DH.Entries = append(creator.DH.Entries, e)
				}
			}
		}
		encodedEntryName, err := govis.Vis(entryPathName, DefaultVisFlags)
		if err != nil {
			return err
		}
		e := Entry{
			Name:   encodedEntryName,
			Pos:    len(creator.DH.Entries),
			Type:   RelativeType,
			Set:    creator.curSet,
			Parent: creator.curDir,
		}
		for _, keyword := range keywords {
			err := func() error {
				var r io.Reader
				if info.Mode().IsRegular() {
					fh, err := creator.fs.Open(path)
					if err != nil {
						return err
					}
					defer fh.Close()
					r = fh
				}
				keyFunc, ok := KeywordFuncs[keyword.Prefix()]
				if !ok {
					return fmt.Errorf("Unknown keyword %q for file %q", keyword.Prefix(), path)
				}
				kvs, err := creator.fs.KeywordFunc(keyFunc)(path, info, r)
				if err != nil {
					return err
				}
				for _, kv := range kvs {
					if kv != "" && !inKeyValSlice(kv, creator.curSet.Keywords) {
						e.Keywords = append(e.Keywords, kv)
					}
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
		if info.IsDir() {
			if creator.curDir != nil {
				creator.curDir.Next = &e
			}
			e.Prev = creator.curDir
			creator.curDir = &e
		} else {
			if creator.curEnt != nil {
				creator.curEnt.Next = &e
			}
			e.Prev = creator.curEnt
			creator.curEnt = &e
		}
		creator.DH.Entries = append(creator.DH.Entries, e)
		return nil
	})
	return creator.DH, err
}

// startWalk walks the file tree rooted at root, calling walkFn for each file or
// directory in the tree, including root. All errors that arise visiting files
// and directories are filtered by walkFn. The files are walked in lexical
// order, which makes the output deterministic but means that for very
// large directories Walk can be inefficient.
// Walk does not follow symbolic links.
func startWalk(c *dhCreator, root string, walkFn filepath.WalkFunc) error {
	info, err := c.fs.Lstat(root)
	if err != nil {
		return walkFn(root, nil, err)
	}
	return walk(c, root, info, walkFn)
}

// walk recursively descends path, calling w.
func walk(c *dhCreator, path string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	err := walkFn(path, info, nil)
	if err != nil {
		if info.IsDir() && err == filepath.SkipDir {
			return nil
		}
		return err
	}

	if !info.IsDir() {
		return nil
	}

	names, err := readOrderedDirNames(c, path)
	if err != nil {
		return walkFn(path, info, err)
	}

	for _, name := range names {
		filename := filepath.Join(path, name)
		fileInfo, err := c.fs.Lstat(filename)
		if err != nil {
			if err := walkFn(filename, fileInfo, err); err != nil && err != filepath.SkipDir {
				return err
			}
		} else {
			err = walk(c, filename, fileInfo, walkFn)
			if err != nil {
				if !fileInfo.IsDir() || err != filepath.SkipDir {
					return err
				}
			}
		}
	}
	c.DH.Entries = append(c.DH.Entries, Entry{
		Name: "..",
		Type: DotDotType,
		Pos:  len(c.DH.Entries),
	})
	if c.curDir != nil {
		c.curDir = c.curDir.Parent
	}
	return nil
}

// readOrderedDirNames reads the directory and returns a sorted list of all
// entries with non-directories first, followed by directories.
func readOrderedDirNames(c *dhCreator, dirname string) ([]string, error) {
	infos, err := c.fs.Readdir(dirname)
	if err != nil {
		return nil, err
	}

	names := []string{}
	dirnames := []string{}
	for _, info := range infos {
		if info.IsDir() {
			dirnames = append(dirnames, info.Name())
			continue
		}
		names = append(names, info.Name())
	}
	sort.Strings(names)
	sort.Strings(dirnames)
	return append(names, dirnames...), nil
}

// signatureEntries is a simple helper function that returns a slice of Entry's
// that describe the metadata signature about the host. Items like date, user,
// machine, and tree (which is specified by argument `root`), are considered.
// These Entry's construct comments in the mtree specification, so if there is
// an error trying to obtain a particular metadata, we simply don't construct
// the Entry.
func signatureEntries(root string) []Entry {
	var sigEntries []Entry
	user, err := user.Current()
	if err == nil {
		userEntry := Entry{
			Type: CommentType,
			Raw:  fmt.Sprintf("#%16s%s", "user: ", user.Username),
		}
		sigEntries = append(sigEntries, userEntry)
	}

	hostname, err := os.Hostname()
	if err == nil {
		hostEntry := Entry{
			Type: CommentType,
			Raw:  fmt.Sprintf("#%16s%s", "machine: ", hostname),
		}
		sigEntries = append(sigEntries, hostEntry)
	}

	if tree := filepath.Clean(root); tree == "." || tree == ".." {
		root, err := os.Getwd()
		if err == nil {
			// use parent directory of current directory
			if tree == ".." {
				root = filepath.Dir(root)
			}
			treeEntry := Entry{
				Type: CommentType,
				Raw:  fmt.Sprintf("#%16s%s", "tree: ", filepath.Clean(root)),
			}
			sigEntries = append(sigEntries, treeEntry)
		}
	} else {
		treeEntry := Entry{
			Type: CommentType,
			Raw:  fmt.Sprintf("#%16s%s", "tree: ", filepath.Clean(root)),
		}
		sigEntries = append(sigEntries, treeEntry)
	}

	dateEntry := Entry{
		Type: CommentType,
		Raw:  fmt.Sprintf("#%16s%s", "date: ", time.Now().Format("Mon Jan 2 15:04:05 2006")),
	}
	sigEntries = append(sigEntries, dateEntry)

	return sigEntries
}

// keywordEntries returns a slice of entries including a comment of the
// keywords requested when generating this manifest.
func keywordEntries(keywords []Keyword) []Entry {
	// Convert all of the keywords to zero-value keyvals.
	return []Entry{
		{
			Type: CommentType,
			Raw:  fmt.Sprintf("#%16s%s", "keywords: ", strings.Join(FromKeywords(keywords), ",")),
		},
	}
}
