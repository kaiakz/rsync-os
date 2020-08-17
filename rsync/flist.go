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
func (L FileList) Diff(R FileList) (new []int, old []int) {
	new = make([]int, 0, len(R))
	old = make([]int, 0, len(L))
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
				new = append(new, i)
			}
			i++
			j++
			break
		case 1:
			old = append(old, j)
			j++
			break
		case -1:
			new = append(new, i)
			i++
			break
		}
	}

	// Handle remains
	for ; i < len(L); i++ {
		old = append(old, i)
	}
	for ; j < len(R); j++ {
		new = append(new, j)
	}

	return
}