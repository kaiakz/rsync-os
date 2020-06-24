package filelist

import (
	"rsync2os/rsync"
)

// Diff two sorted list
// Return two lists: new files, deleted files
func (cache *Cache) Diff(list *rsync.FileList) (*rsync.FileList, *rsync.FileList) {

	// Interate cache.module(A) & list(B), both A & B must be sorted lexicographically before

	// Compare their path

	// bytes.Compare

	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b

	// If > 0, B doesn't have

	// If == 0, A & B have

	// If < 0, A doesn't have

	return nil, nil
}