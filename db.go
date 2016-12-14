package sled

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Avalanche-io/sled/storage"
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

type tx struct {
	t string
	k string
	v interface{}
}

func (t *tx) Action() string {
	return t.t
}

func (t *tx) Key() string {
	return t.k
}

func (t *tx) Value() interface{} {
	return t.v
}

// Open the sled database at the provided path, or create a new one.
func (s *Sled) Open(filepath *string) error {

	if s.db != nil {
		return dbOpenError
	}

	s.file_ch = make(chan Tx)
	db_ch := make(chan Tx)
	s.err_ch = make(chan error)

	if filepath == nil {
		go storage_thread_noop(s.st, s.file_ch, db_ch, s.err_ch)
		go db_thread_noop(s.db, db_ch, s.err_ch)
		return nil
	}

	db, err := bolt.Open(*filepath, 0777, nil)
	if err != nil {
		return SledError(err.Error())
	}
	s.db = db
	wg := sync.WaitGroup{}
	s.close_wg = &wg

	go storage_thread(s.st, s.file_ch, db_ch, s.err_ch)
	go db_thread(s.db, db_ch, s.err_ch)

	s.createBuckets()
	s.loadAllKeys()
	return nil
}

func storage_thread(st storage.IO, tx_in <-chan Tx, out chan<- Tx, ech chan<- error) {
	for t := range tx_in {
		var err error
		var id *asset.ID
		key := t.Key()
		value := t.Value()
		action := t.Action()

		switch action {
		case "save":
			id, err = st.Save(value)
			if err != nil {
				ech <- err
				break
			}
			out <- &tx{"save", key, id}
		case "delete":
			out <- &tx{"delete", key, nil}
			// don't delete files yet.
		}
	}
	close(out)
}

func db_thread(db *bolt.DB, tx_in <-chan Tx, ech chan<- error) {
	for t := range tx_in {
		var err error
		var id []byte
		key := []byte(t.Key())
		value := t.Value()
		action := t.Action()

		switch value.(type) {
		case nil:
			// do nothing
		case *asset.ID:
			id = value.(*asset.ID).RawBytes()
		default:
			if action == "save" {
				ech <- SledError("DB received non ID value.")
			}
			continue
		}

		switch action {
		case "save":
			err = db.Batch(func(btx *bolt.Tx) error {
				b := btx.Bucket([]byte("assets"))
				return b.Put(key, id)
			})
		case "delete":
			err = db.Batch(func(btx *bolt.Tx) error {
				b := btx.Bucket([]byte("assets"))
				return b.Delete(key)
			})
		}

		if err != nil {
			ech <- SledError(err.Error())
		}
	}
	close(ech)
}

func storage_thread_noop(st storage.IO, tx_in <-chan Tx, out chan<- Tx, ech chan<- error) {
	for range tx_in {
		out <- &tx{"", "", nil}
		// noop
	}
	close(out)
}

func db_thread_noop(db *bolt.DB, tx_in <-chan Tx, ech chan<- error) {
	for range tx_in {
		// noop
	}
	close(ech)
}

// Wait for database operations to complete, and close database.
func (s *Sled) Close() error {
	close(s.file_ch)
	s.Wait()

	<-s.err_ch

	err := s.db.Close()
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
	for ele := range s.db_iterator("assets", nil, nil) {
		p := sledPointer{ele.Id()}
		s.ct.Insert(ele.Key(), &p)
	}
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
