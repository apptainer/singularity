
#ifndef __CLEAN_DIR_H

// Recursively remove all files within a directory.
// Does not follow symlinks and does not cross to different
// devices.
// Returns non-zero on errors.
int clean_dir(const char *);

#endif  // __CLEAN_DIR_H
