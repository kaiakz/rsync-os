package filelist

import (
	"fmt"
	"log"
	"rsync2os/rsync"

	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

type Cache struct {
	db      *bolt.DB
	module  *bolt.Bucket
	prepath string
}

func Open(module string, prepath string) *Cache {
	db, err := bolt.Open("r.db", 0666, nil)
	if err != nil {
		return nil
	}
	tx, err := db.Begin(true)
	if tx == nil {
		return nil
	}
	defer tx.Rollback()

	var bucket *bolt.Bucket
	bucket, err = tx.CreateBucketIfNotExists([]byte(module))
	if err != nil {
		return nil
	}
	return &Cache{
		db:      db,
		module:  bucket,
		prepath: prepath,
	}
}

func (cache *Cache) Put(info *rsync.FileInfo) error {
	key := []byte(cache.prepath + info.Path)
	value, err := proto.Marshal(&FInfo{
		Size:  info.Size,
		Mtime: info.Mtime,
		Mode:  info.Mode,
	})
	if err != nil {
		return err
	}
	return cache.module.Put(key, value)
}

func (cache *Cache) Get(key []byte) *FInfo {
	value := cache.module.Get(key)
	if value != nil {
		info := &FInfo{}
		err := proto.Unmarshal(value, info)
		if err == nil {
			return info
		}
	}
	return nil
}

func (cache *Cache) PutAll(list *rsync.FileList) error {
	for _, info := range *list {
		err := cache.Put(&info)
		if err != nil {
			return err
		}
	}
	return nil
}

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
			fmt.Println(m)
		}

		return nil
	})

	if err != nil {

	}
}
