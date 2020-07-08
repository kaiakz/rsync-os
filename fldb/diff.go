package fldb

import (
	"bytes"
	"rsync-os/rsync"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// Diff two sorted list
// Return two lists: new files, delete files
func (cache *Cache) Diff(list rsync.FileList) ([]int, [][]byte) {

	downloadList := make([]int, 0, 4096)
	deleteList := make([][]byte, 0, 4096)
	// Iterate cache.module(A) & list(B), both A & B must be sorted lexicographically before
	err := cache.db.View(func(tx *bolt.Tx) error {
		mod := tx.Bucket(cache.module)
		// If bucket does not exist, create the bucket
		if mod == nil {
			var err error
			mod, err = tx.CreateBucket(cache.module)
			if err != nil {
				return err
			}
		}

		c := mod.Cursor()
		prefix := cache.prepath
		i := 0
		k, v := c.Seek(prefix)
		for i < list.Len() && k != nil && bytes.HasPrefix(k, prefix) {
			// The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
			// Compare their paths by bytes.Compare
			// The result will be 0 if a==b, -1 if a < b, and +1 if a > b
			// If 1, B doesn't have
			// If 0, A & B have
			// If -1, A doesn't have
			switch bytes.Compare([]byte(list[i].Path), k) {
			case 0:
				info := &FInfo{}
				err := proto.Unmarshal(v, info)
				if err != nil {
					// TODO
				}
				if list[i].Mtime != info.Mtime || list[i].Size != info.Size {
					downloadList = append(downloadList, i)
				}
				i++
				k, v = c.Next()
				break
			case 1:
				deleteList = append(deleteList, k)
				k, v = c.Next()
				break
			case -1:
				downloadList = append(downloadList, i)
				i++
				break
			}
		}

		// Handle remains
		if i == list.Len() {
			for ; k != nil && bytes.HasPrefix(k, prefix); k, _ = c.Next() {
				deleteList = append(deleteList, k)
			}
		} else {
			for ; i < list.Len(); i++ {
				downloadList = append(downloadList, i)
			}
		}

		return nil
	})
	if err != nil {

	}

	return downloadList[:], deleteList[:]
}
