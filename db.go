package sled

import (
	"encoding/json"
	"errors"
	"github.com/boltdb/bolt"
	"log"
	"sync"
)

var bucket_names []string

func init() {
	bucket_names = []string{"assets", "attributes"}
}

type createBucketError string

func (e createBucketError) Error() string {
	return "Error creating bucket: " + string(e)
}

func (s *Sled) Open(path string) error {
	db, err := bolt.Open(path, 0777, nil)
	if err != nil {
		return err
	}
	s.db = db
	wg := sync.WaitGroup{}
	s.close_wg = &wg
	return nil
}

func (s *Sled) loadAllKeys() {
	s.loading = make(chan struct{})
	s.close_wg.Add(1)
	go func() {
		defer s.close_wg.Done()
		for ele := range s.db_iterator("assets", nil, nil) {
			s.ct.Insert(ele.Key(), ele.Value())
		}
		close(s.loading)
	}()
}

type element struct {
	key   []byte
	value interface{}
}

func (k *element) Key() []byte {
	return k.key
}

func (k *element) Value() interface{} {
	return k.value
}

func (s *Sled) db_iterator(bucket string, key []byte, cancel <-chan struct{}) <-chan element {
	out := make(chan element)
	go func() {
		err := s.db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(bucket))
			if b == nil {
				return errors.New("No bucket " + bucket)
			}
			c := b.Cursor()
			var k, v []byte
			if key == nil {
				k, v = c.First()
			} else {
				k, v = c.Seek(key)
			}

			for ; k != nil; k, v = c.Next() {
				var ent element
				if v == nil || len(v) == 0 {
					ent = element{k, nil}
				} else {
					var val interface{}
					err := json.Unmarshal(v, &val)
					if err != nil {
						log.Fatalf("db.Iterator error: k: %v, \tv: %v\n", k, v)
						return err
					}
					ent = element{k, val}
				}
				select {
				case out <- ent:
				case <-cancel:
					return nil
				}
			}
			return nil
		})
		if err != nil {
			panic(err)
		}
		close(out)
	}()
	return out
}

// creates db buckets
func (s *Sled) createBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, name := range bucket_names {
			_, err := tx.CreateBucketIfNotExists([]byte(name))
			if err != nil {
				return createBucketError(name + err.Error())
			}
		}
		return nil
	})
}

// put sets the value of a key for a given bucket
func (s *Sled) put_db(db *bolt.DB, bucket string, key []byte, data []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Put(key, data)
	})
}

// get retrieves the value a key for the given bucket
func (s *Sled) load_db(db *bolt.DB, bucket string, key []byte) ([]byte, error) {
	var data []byte
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data = b.Get(key)
		return nil
	})
	return data, err
}
