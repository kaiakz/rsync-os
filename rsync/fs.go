package rsync

import (
	"io"
)

type FileMetadata struct {
	Mtime int32
	Mode  FileMode
}

// File System: need to handle all type of files: regular, folder, symlink, etc
type FS interface {
	Put(fileName string, content io.Reader, fileSize int64, metadata FileMetadata) (written int64, err error)
	//Get(fileName string, metadata FileMetadata)
	Delete(fileName string) error
	List() (FileList, error)
}