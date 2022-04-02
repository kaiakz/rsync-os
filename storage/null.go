package storage

import (
	"io"
	"rsync-os/rsync"
)

/*
A /dev/null-like storage backend for testing

*/

type NULL struct {
}

func (nu *NULL) Write(p []byte) (n int, err error) {
	// Do nothing
	return len(p), nil
}

func (nu *NULL) Put(fileName string, content io.Reader, fileSize int64, metadata rsync.FileMetadata) (written int64, err error) {
	// We can do some log here
	return io.Copy(nu, content)
}

func (nu *NULL) Delete(fileName string, mode rsync.FileMode) error {
	return nil
}

func (nu *NULL) List() (rsync.FileList, error) {
	return nil, nil
}
