package cache

import (
	"log"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/kaiakz/rsync-os/rsync"
	bolt "go.etcd.io/bbolt"
)

// Test
func Snapshot(list rsync.FileList, module string, prepath string) {
	startTime := time.Now()
	db, err := bolt.Open("test.db", 0666, nil)
	defer db.Close()
	if err != nil {
		return
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(module))
		// If bucket does not exist, create the bucket
		if bucket == nil {
			var err error
			bucket, err = tx.CreateBucket([]byte(module))
			if err != nil {
				return err
			}
		}
		for _, info := range list {
			key := append([]byte(prepath), info.Path[:]...)
			value, err := proto.Marshal(&FInfo{
				Size:  info.Size,
				Mtime: info.Mtime,
				Mode:  int32(info.Mode), // FIXME: convert uint32 to int32
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
	log.Println("Saved Duration", time.Since(startTime))
}
