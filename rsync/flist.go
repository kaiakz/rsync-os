package rsync

import (
	"bytes"
	"os"
)

type FileInfo struct {
	Path  []byte
	Size  int64
	Mtime int32
	Mode  os.FileMode
}

// For unix/linux
type FileMode uint32

func (m FileMode) IsREG() bool {
	return (m&S_IFMT)==S_IFREG
}

func (m FileMode) IsDIR() bool {
	return (m&S_IFMT)==S_IFDIR
}

func (m FileMode) IsBLK() bool {
	return (m&S_IFMT)==S_IFBLK
}

func (m FileMode) IsLNK() bool {
	return (m&S_IFMT)==S_IFLNK
}

func (m FileMode) IsFIFO() bool {
	return (m&S_IFMT)==S_IFIFO
}

func (m FileMode) IsSock() bool {
	return (m&S_IFMT)==S_IFSOCK
}

// Return only unix permission bits
func (m FileMode) Perm() FileMode {
	return m&0777
}

// Convert to os.FileMode
func (m FileMode) Convert() os.FileMode {
	mode := os.FileMode(m & 0777)
	switch m & S_IFMT {
	case S_IFREG:
		// For regular files, none will be set.
		break
	case S_IFDIR:
		mode |= os.ModeDir
		break
	case S_IFLNK:
		mode |= os.ModeSymlink
		break
	case S_IFBLK:
		mode |= os.ModeDevice
		break
	case S_IFSOCK:
		mode |= os.ModeSocket
		break
	case S_IFIFO:
		mode |= os.ModeNamedPipe
		break
	case S_IFCHR:
		mode |= os.ModeCharDevice
		break
	default:
		mode |= os.ModeIrregular
		break
	}
	return mode
}

// strmode
func (m FileMode) String() string {
	chars := []byte("-rwxrwxrwx")
	switch m & S_IFMT {
	case S_IFREG:
		break
	case S_IFDIR:
		chars[0] = 'd'
		break
	case S_IFLNK:
		chars[0] = 'l'
		break
	case S_IFBLK:
		chars[0] = 'b'
		break
	case S_IFSOCK:
		chars[0] = 's'
		break
	case S_IFIFO:
		chars[0] = 'p'
		break
	case S_IFCHR:
		chars[0] = 'c'
		break
	default:
		chars[0] = '?'
		break
	}
	for i:=0; i < 9; i++ {
		if m & (1 << i) == 0 {
			chars[9-i] = '-'
		}
	}
	return string(chars)
}

type FileList []FileInfo

func (L FileList) Len() int {
	return len(L)
}

func (L FileList) Less(i, j int) bool {
	if bytes.Compare(L[i].Path, L[j].Path) == -1 {
		return true
	}
	return false
}

func (L FileList) Swap(i, j int) {
	L[i], L[j] = L[j], L[i]
}

/* Diff two sorted rsync file list, return their difference
list NEW: only R has.
list OLD: only L has.
 */
func (L FileList) Diff(R FileList) (newitems []int, olditems []int) {
	newitems = make([]int, 0, len(R))
	olditems = make([]int, 0, len(L))
	i := 0	// index of L
	j := 0	// index of R

	for i < len(L) && j < len(R) {
		// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
		// Compare their paths by bytes.Compare
		// The result will be 0 if a==b, -1 if a < b, and +1 if a > b
		// If 1, B doesn't have
		// If 0, A & B have
		// If -1, A doesn't have
		switch bytes.Compare(L[i].Path, R[j].Path) {
		case 0:
			if L[i].Mtime != R[j].Mtime || L[i].Size != R[j].Size {
				newitems = append(newitems, i)
			}
			i++
			j++
			break
		case 1:
			olditems = append(olditems, j)
			j++
			break
		case -1:
			newitems = append(newitems, i)
			i++
			break
		}
	}

	// Handle remains
	for ; i < len(L); i++ {
		olditems = append(olditems, i)
	}
	for ; j < len(R); j++ {
		newitems = append(newitems, j)
	}

	return
}