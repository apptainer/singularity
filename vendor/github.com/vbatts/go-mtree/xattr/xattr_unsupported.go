// +build !linux

package xattr

// Get would return the extended attributes, but this unsupported feature
// returns nil, nil
func Get(path, name string) ([]byte, error) {
	return nil, nil
}

// Set would set the extended attributes, but this unsupported feature returns
// nil
func Set(path, name string, value []byte) error {
	return nil
}

// List would return the keys of extended attributes, but this unsupported
// feature returns nil, nil
func List(path string) ([]string, error) {
	return nil, nil
}
