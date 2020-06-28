package fldb

import (
	"bytes"
	"rsync-os/rsync"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

// Diff two sorted list
// Return two lists: new files, deleted files
func (cache *Cache) Diff(list *rsync.FileList) {

	db, err := bolt.Open("test.db", 0666, nil)
	if err != nil {
		return
	}

	defer db.Close()

	// Iterate cache.module(A) & list(B), both A & B must be sorted lexicographically before

	i := 0
	downloadList := make([]int, 1000)
	deleteList := make([]string, 1000)
	db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		c := tx.Bucket([]byte("MyBucket")).Cursor()

		prefix := []byte(cache.prepath)
		// for k, v := c.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = c.Next() {

		// }
		k, v := c.Seek(prefix)
		for i < list.Len() && k != nil && bytes.HasPrefix(k, prefix) {
			switch bytes.Compare([]byte((*list)[i].Path), k) {
			case 0:
				info := &FInfo{}
				err := proto.Unmarshal(v, info)
				if err != nil {
				}
				if (*list)[i].Mtime != info.Mtime || (*list)[i].Size != info.Size {
					downloadList = append(downloadList, i)
				}
				i++
				k, v = c.Next()
				break
			case 1:
				deleteList = append(deleteList, string(k))
				k, v = c.Next()
				break
			case -1:
				downloadList = append(downloadList, i)
				i++
				break
			}
		}

		return nil
	})

	// Compare their path

	// bytes.Compare

	// The result will be 0 if a==b, -1 if a < b, and +1 if a > b

	// If > 0, B doesn't have

	// If == 0, A & B have

	// If < 0, A doesn't have

}
