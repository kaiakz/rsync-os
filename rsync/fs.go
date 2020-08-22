package rsync

import (
	"io"
	"os"
)

type FileMetadata struct {
	Mtime int32
	Mode  os.FileMode
}

// File System
type FS interface {
	Put(fileName string, content io.Reader, fileSize int64, metadata FileMetadata) (written int64, err error)
	//Get(fileName string, metadata FileMetadata)
	Delete(fileName string) error
	List() (FileList, error)
}