package fldb

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	bolt "go.etcd.io/bbolt"
	"log"
	"rsync-os/rsync"
)

// Test
func Save(list *rsync.FileList, module string, prepath string) {
	db, err := bolt.Open("test.db", 0666, nil)
	if err != nil {
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(module))
		if err != nil {
			log.Panicln("create module as bucket failed", err)
			return err
		}
		for _, info := range *list {
			key := []byte(prepath + info.Path)
			value, err := proto.Marshal(&FInfo{
				Size:  info.Size,
				Mtime: info.Mtime,
				Mode:  info.Mode,
			})
			if err != nil {
				log.Println("Marshal failed", err)
				return err
			}
			bucket.Put(key, value)
		}
		return nil
	})
	if err != nil {
		log.Println("Update failed", err)
	}

	err = db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(module))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			fmt.Println("key= ", string(k))
			var m FInfo
			proto.Unmarshal(v, &m)
			fmt.Println(m.GetMtime(), m.GetSize())
		}

		return nil
	})

	if err != nil {

	}
}