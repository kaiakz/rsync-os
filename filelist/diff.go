package filelist

import "rsync2os/rsync"

// Return two lists: new files, deleted files
func (cache *Cache) Diff(list *rsync.FileList) (*rsync.FileList, *rsync.FileList) {
	return nil, nil
}