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
	Put(objectName string, reader io.Reader, objectSize int64, metadata FileMetadata) (n int64, err error)
	//Get(fileName string, metadata FileMetadata)
	Delete(objectName string) error
	List() FileList
}