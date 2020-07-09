package fldb

import (
	"github.com/golang/protobuf/proto"
	bolt "go.etcd.io/bbolt"
	"log"
	"rsync-os/rsync"
)

// Test
func Snapshot(list rsync.FileList, module string, prepath string) {
	db, err := bolt.Open("test.db", 0666, nil)
	defer db.Close()
	if err != nil {
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(module))
		if err != nil {
			log.Panicln("create module as bucket failed", err)
			return err
		}
		for _, info := range list {
			key := append([]byte(prepath), info.Path[:]...)
			value, err := proto.Marshal(&FInfo{
				Size:  info.Size,
				Mtime: info.Mtime,
				Mode:  info.Mode,
			})
			if err != nil {
				log.Println("Marshal failed", err)
				return err
			}
			err = bucket.Put(key, value)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Update failed", err)
	}
}