package filelist

import (
	"google.golang.org/protobuf/proto"
	"rsync2os/rsync"
	bolt "go.etcd.io/bbolt"
)

type Cache struct {
	db *bolt.DB
	module *bolt.Bucket
	prepath string
}

func Open(module string, prepath string) {
	db, err := bolt.Open("r.db", 0666, nil)
	if err != nil {
		//return err
	}
	tx, err := db.Begin(true)
	defer tx.Rollback()

	_, err = tx.CreateBucketIfNotExists([]byte(module))
	if err := tx.Commit(); err != nil {

	}
}

func (cache *Cache) Put(info *rsync.FileInfo) error {
	key := []byte(cache.prepath+info.Path)
	value, err := proto.Marshal(&FInfo{
		Size: info.Size,
		Mtime: info.Mtime,
		Mode: info.Mode,
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

func Save(list *rsync.FileList) {

}
