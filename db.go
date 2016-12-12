package sled

import (
	"errors"
	"fmt"
	"sync"

	"github.com/boltdb/bolt"
	"github.com/etcenter/c4/asset"
)

var bucket_names []string

func init() {
	bucket_names = []string{"assets", "attributes"}
}

type createBucketError string

func (e createBucketError) Error() string {
	return "Error creating bucket: " + string(e)
}

// Open the sled database at the provided path, or create a new one.
func (s *Sled) Open(filepath string) error {
	if s.db != nil {
		return dbOpenError
	}
	if filepath == "/tmp/sled.db" {
		panic("tried to open /tmp/sled.db")
	}

	db, err := bolt.Open(filepath, 0777, nil)
	if err != nil {
		return SledError(err.Error())
	}
	s.db = db
	wg := sync.WaitGroup{}
	s.close_wg = &wg

	if err != nil {
		return SledError(err.Error())
	}

	s.createBuckets()
	s.loadAllKeys()
	return nil
}

// Wait for database operations to complete, and close database.
func (s *Sled) Close() error {
	s.Wait()
	err := s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("assets"))
		for ele := range s.ct.Iterator(nil) {
			p := b.Get(ele.Key)
			if p == nil {
				id, err := s.st.Save(ele.Value)
				if err != nil {
					return err
				}
				err = b.Put(ele.Key, id.RawBytes())
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	err = s.db.Close()
	s.db = nil
	return err
}

// Wait for database operations to complete.
func (s *Sled) Wait() {
	if s.close_wg != nil {
		s.close_wg.Wait()
	}
}

func (s *Sled) loadAllKeys() {
	s.loading = make(chan struct{})
	s.close_wg.Add(1)
	go func() {
		defer s.close_wg.Done()
		for ele := range s.db_iterator("assets", nil, nil) {
			s.ct.Insert(ele.Key(), ele.Id())
		}
		close(s.loading)
	}()
}

type element struct {
	key []byte
	id  *asset.ID
}

func (k *element) Key() []byte {
	return k.key
}

func (k *element) Id() *asset.ID {
	return k.id
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
					if len(v) != 64 {
						return errors.New(fmt.Sprintf("sled.DB error: Wrong length for stored value: %d", len(v)))
					}
					id := asset.BytesToID(v)
					ent = element{k, id}
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
func (s *Sled) put_db(bucket string, key string, id *asset.ID) error {
	if id == nil {
		return errors.New("Id is nil")
	}
	return s.db.Batch(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Put([]byte(key), id.RawBytes())
	})
}

// get retrieves the value for a key for a given bucket
func (s *Sled) get_db(bucket string, key string) (id *asset.ID, err error) {
	var data []byte
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data = b.Get([]byte(key))
		if data == nil {
			return errors.New("db: No such key. " + key)
		}
		if len(data) != 64 {
			return errors.New("Value stored in database is not a C4 ID.")
		}
		id = asset.BytesToID(data)
		return nil
	})
	return
}
