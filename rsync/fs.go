package rsync

import (
	"errors"
	"io"
)

type FileMetadata struct {
	Mtime int32
	Mode  FileMode
}

// File System: need to handle all type of files: regular, folder, symlink, etc
type FS interface {
	Put(fileName string, content io.Reader, fileSize int64, metadata FileMetadata) (written int64, err error)
	//Get(fileName string, metadata FileMetadata) (File, error)
	Delete(fileName string, mode FileMode) error
	List() (FileList, error)
	//Stats() (seekable bool)
}

// Interface: Read, ReadAt, Seek, Close
// type File interface {
// 	io.Reader
// 	io.ReaderAt
// 	io.Seeker
// 	io.Closer
// }

type ReceivingFile struct {
	src       *Conn
	remaining int
	chksum    [16]byte
}

func NewReceivingFile(src *Conn) *ReceivingFile {
	return &ReceivingFile{
		src:       src,
		remaining: 0,
	}
}

func (f *ReceivingFile) Read(p []byte) (n int, err error) {
	if f.remaining == 0 {
		err = f.getToken()
		if err != nil {
			return
		}
	}
	l := len(p)
	if l <= f.remaining {
		n, err = f.src.Read(p)
	} else {
		n, err = f.src.Read(p[:f.remaining])
	}
	f.remaining -= n
	return
}

func (f *ReceivingFile) getToken() (err error) {
	var token int32
	if token, err = f.src.ReadInt(); err != nil {
		return
	}
	if token == 0 {
		err = io.EOF
	} else if token < 0 {
		err = errors.New("Block Checksum hasn't supported yet")
	} else {
		f.remaining = int(token)
	}
	return
}

func (f ReceivingFile) Reset() {
	f.remaining = 0
}
