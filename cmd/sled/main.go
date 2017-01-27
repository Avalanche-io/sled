package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/boltdb/bolt"

	"github.com/Avalanche-io/sled"
)

var (
	// go build -ldflags "-X main.GitId=`git rev-parse --short=7 HEAD`"
	GitId string
)

func main() {
	os.Exit(sled_main())
}

func sled_main() int {
	// log.SetHandler(text.New(os.Stderr))
	log.SetHandler(cli.New(os.Stderr))
	logger := log.WithFields(log.Fields{
		"app":   "sled",
		"build": GitId,
	})
	args := os.Args[1:]
	if len(args) == 0 || (len(args)%2 != 0 && len(args) != 1) {
		logger.Fatalf("Usage:\tsled key value\n")
		return -1
	}

	sl := sled.New()
	bucket := []byte("default")
	db, err := bolt.Open("sled.db", 0600, nil)
	if err != nil {
		logger.WithError(err).Error("unable to open database file")
		return -1
	}
	defer db.Close()
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	})
	if err != nil {
		logger.WithError(err).Error("unable to create bucket")
		return -1
	}
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		return b.ForEach(func(k, data []byte) error {
			var v interface{}
			err := json.Unmarshal(data, &v)
			if err != nil {
				logger.WithField("data", string(data)).WithError(err)
				return err
			}
			return sl.Set(string(k), v)
		})
	})

	if err != nil {
		logger.WithError(err).Errorf("unable to read data from db")
		return -1
	}

	if len(args) == 1 {
		var v string
		err := sl.Get(args[0], &v)
		if err != nil {
			logger.WithError(err).Errorf("unable to get value for key \"%s\"", args[0])
			return -1
		}
		fmt.Println(v)
		return 0
	}

	for i := 0; i < len(args); i += 2 {
		err := sl.Set(args[i], args[i+1])
		if err != nil {
			logger.WithError(err).Errorf("unable to set value for key \"%s\"", args[i])
			return -1
		}
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		for elm := range sl.Iterate(nil) {
			data, err := json.Marshal(elm.Value())
			if err != nil {
				logger.WithField(elm.Key(), elm.Value())
				elm.Close()
				return err
			}
			err = b.Put([]byte(elm.Key()), data)
			if err != nil {
				logger.WithField(elm.Key(), elm.Value())
				elm.Close()
				return err
			}
			elm.Close()
		}
		return nil
	})
	if err != nil {
		logger.WithError(err).Error("unable to save to db")
		return -1
	}

	return 0
}
