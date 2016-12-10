package sled_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
	"github.com/Avalanche-io/sled/config"
)

func TestDBOpen(t *testing.T) {
	t.Log("init")
	is := is.New(t)

	t.Log("do")
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestConfiguration_")
	is.NoErr(err)
	defer os.RemoveAll(dir)
	cfg := config.New().WithRoot(dir).WithDB("db.sled")
	sl := sled.New(cfg)
	is.NotNil(cfg)
	is.NotNil(sl)

	t.Log("check")
	err = sl.Open("db2.sled")
	is.Equal(err.Error(), "Sled error: Db already open.")
}
