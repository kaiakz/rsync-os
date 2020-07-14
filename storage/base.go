package storage

import (
	"io"
	"rsync-os/rsync"
)

type FileMetadata struct {
	Mtime int32
	Mode  int32
}

type IO interface {
	Write(fileName string, reader io.Reader, fileSize int64, metadata FileMetadata) (n int64, err error)
	//Read(fileName string, metadata FileMetadata)
	Delete(fileName string) error
	List() rsync.FileList
}
