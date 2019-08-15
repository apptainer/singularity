package mtree

// dhCreator is used in when building a DirectoryHierarchy
type dhCreator struct {
	DH     *DirectoryHierarchy
	fs     FsEval
	curSet *Entry
	curDir *Entry
	curEnt *Entry
}
