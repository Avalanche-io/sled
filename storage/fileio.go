package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	"github.com/Avalanche-io/sled/config"
	"github.com/etcenter/c4/asset"
)

type IO interface {
	Save(value interface{}) (id *asset.ID, err error)
	Load(id *asset.ID) (interface{}, error)
	SizeOf(id *asset.ID) (int64, error)
}

// fileIO holds the io context, but is not exported to compel use of the 'New' constructor.
type fileIO struct {
	cfg *config.Config
}

// Create a new storage context
func New(cfg *config.Config) *fileIO {
	return &fileIO{cfg}
}

func isC4ID(data []byte) bool {
	valid_len := len(data)%90 == 0
	valid_prefix := string(data[:2]) == "c4"
	// has no incorrect characters
	return valid_len && valid_prefix
}

func notJSON(b []byte) bool {
	var js map[string]interface{}
	// Do we also have to check if this is a json array?
	return json.Unmarshal(b, &js) != nil
}

func getString(data []byte) *string {
	var s string
	if json.Unmarshal(data, &s) != nil {
		return nil
	}
	return &s
}

func BytesToAny(data []byte) (value interface{}, err error) {
	value = nil
	err = nil

	switch {
	case len(data) == 0:
		return
	case isC4ID(data):
		var ids asset.IDSlice

		for len(data) >= 90 {
			id := asset.BytesToID(data[:90])
			ids.Push(id)
		}
		if len(ids) == 1 {
			value = ids[0]
		} else {
			value = ids
		}
		return
	case notJSON(data):
		strp := getString(data)
		if strp != nil {
			value = *strp
		} else {
			value = data
		}
	default:
		err = json.Unmarshal(data, value)
		return
	}
	return
}

func AnyToBytes(value interface{}) (data []byte, err error) {
	switch value.(type) {
	// case *string:
	// 	data = []byte(*(value.(*string)))
	// case string:
	// 	data = []byte("\"" + value.(string) + "\"")
	case []byte:
		data = value.([]byte)
	default:
		data, err = json.Marshal(value)
		if err != nil {
			return
		}
	}
	return
}

// Save
func (ctx *fileIO) Save(value interface{}) (id *asset.ID, err error) {
	var data []byte

	data, err = AnyToBytes(value)
	if err != nil {
		return
	}

	id, err = asset.Identify(bytes.NewReader(data))
	if err != nil {
		return
	}

	err = ctx.saveBytes(config.NewFilename(id.String()), data)
	return
}

func exists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (ctx *fileIO) saveBytes(name *config.Filename, data []byte) error {
	filepath := ctx.cfg.FilePath(name)

	if exists(filepath) {
		return nil
	}

	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)

	return err
}

func (ctx *fileIO) loadBytes(name *config.Filename) ([]byte, error) {
	filepath := ctx.cfg.FilePath(name)

	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, errors.New("storage.Load: Unable to stat file. " + err.Error())
	}

	data := make([]byte, info.Size())
	_, err = f.Read(data)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return data, nil
}

func (ctx *fileIO) Load(id *asset.ID) (interface{}, error) {
	data, err := ctx.loadBytes(config.NewFilename(id.String()))
	if err != nil {
		return nil, err
	}
	var val interface{}
	val, err = BytesToAny(data)
	// err = json.Unmarshal(data, &val)
	if err != nil {
		return nil, errors.New("storage.Load: Failed to unmarshal data. " + err.Error())
	}
	return val, nil
}

func (ctx *fileIO) SizeOf(id *asset.ID) (int64, error) {
	filepath := ctx.cfg.FilePath(config.NewFilename(id.String()))
	f, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}
