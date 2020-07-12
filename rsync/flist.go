package rsync

import "bytes"

type FileInfo struct {
	Path  []byte
	Size  int64
	Mtime int32
	Mode  int32
}

type FileList []FileInfo

func (I FileList) Len() int {
	return len(I)
}

func (I FileList) Less(i, j int) bool {
	if bytes.Compare(I[i].Path, I[j].Path) == -1 {
		return true
	}
	return false
}

func (I FileList) Swap(i, j int) {
	I[i], I[j] = I[j], I[i]
}
