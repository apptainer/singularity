package mtree

// Check a root directory path against the DirectoryHierarchy, regarding only
// the available keywords from the list and each entry in the hierarchy.
// If keywords is nil, the check all present in the DirectoryHierarchy
//
// This is equivalent to creating a new DirectoryHierarchy with Walk(root, nil,
// keywords, fs) and then doing a Compare(dh, newDh, keywords).
func Check(root string, dh *DirectoryHierarchy, keywords []Keyword, fs FsEval) ([]InodeDelta, error) {
	if keywords == nil {
		keywords = dh.UsedKeywords()
	}

	newDh, err := Walk(root, nil, keywords, fs)
	if err != nil {
		return nil, err
	}

	return Compare(dh, newDh, keywords)
}

// TarCheck is the tar equivalent of checking a file hierarchy spec against a
// tar stream to determine if files have been changed. This is precisely
// equivalent to Compare(dh, tarDH, keywords).
func TarCheck(tarDH, dh *DirectoryHierarchy, keywords []Keyword) ([]InodeDelta, error) {
	if keywords == nil {
		return Compare(dh, tarDH, dh.UsedKeywords())
	}
	return Compare(dh, tarDH, keywords)
}
