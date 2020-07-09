package fldb

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	bolt "go.etcd.io/bbolt"
)

func IterBucket(module string) {
	db, err := bolt.Open("test.db", 0666, nil)
	defer db.Close()
	if err != nil {
		return
	}
	err = db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(module))

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m FInfo
			proto.Unmarshal(v, &m)
			fmt.Println(string(k), m.GetMtime(), m.GetSize())
		}

		return nil
	})

	if err != nil {

	}
}
