package rsync

import (
	"encoding/binary"
	"hash"
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

// An error brings the block offset
type CopyingBlock struct {
	Offset int
}

func (b *CopyingBlock) Error() string {
	return "Copying a block:" + string(b.Offset)
}

type ReceivingFile struct {
	src       *Conn
	seed      int32
	remaining int
	chksum    hash.Hash
}

func NewReceivingFile(src *Conn, seed int32) *ReceivingFile {
	return &ReceivingFile{
		src:       src,
		seed:      seed,
		remaining: 0,
	}
}

func (f *ReceivingFile) Read(p []byte) (n int, err error) {
	if f.remaining == 0 {
		err = f.parseToken()
		if err != nil {
			return
		}
	}
	if len(p) <= f.remaining {
		n, err = f.src.Read(p)
		if err == nil {
			f.chksum.Write(p)
		}
	} else {
		sp := p[:f.remaining]
		n, err = f.src.Read(sp)
		if err == nil {
			f.chksum.Write(sp)
		}
	}
	f.remaining -= n
	return
}

/* If here comes a negative token(copy a block from the local file),
we need to notify the caller to handle it.
*/
func (f *ReceivingFile) parseToken() (err error) {
	var token int32
	if token, err = f.src.ReadInt(); err != nil {
		return
	}
	if token == 0 { // the end of the file
		err = io.EOF
	} else if token < 0 {
		err = &CopyingBlock{
			Offset: int(token),
		}
	} else {
		f.remaining = int(token)
	}
	return
}

func (f *ReceivingFile) Reset() {
	f.remaining = 0
	f.chksum.Reset()
	binary.Write(f.chksum, binary.LittleEndian, f.seed)
}

func (f *ReceivingFile) Sum() []byte {
	return f.chksum.Sum(nil)
}
