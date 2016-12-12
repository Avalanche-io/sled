package sled_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
	"github.com/Avalanche-io/sled/config"

	"github.com/etcenter/c4/asset"
)

func TestConfiguration(t *testing.T) {
	t.Log("init")
	is := is.New(t)

	t.Log("do")
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestConfiguration_")
	is.NoErr(err)
	defer os.RemoveAll(dir)
	cfg := config.New().WithRoot(dir).WithDB("db.sled")
	sl := sled.New(cfg)

	t.Log("check")
	is.NotNil(cfg)
	is.NotNil(sl)

	if _, err := os.Stat(dir + "/db.sled"); os.IsNotExist(err) {
		is.Fail("Db file not created")
	}
}

func TestReadWrite(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestReadWrite_")
	is.NoErr(err)
	defer os.RemoveAll(dir)
	cfg := config.New().WithRoot(dir).WithDB("sled.db")

	t.Log("do")
	sl := sled.New(cfg)
	sl.Set("foo", "bar")
	value, err := sl.Get("foo")

	t.Log("check")
	is.NoErr(err)
	is.Equal(value.(string), "bar")
}

func TestCreatesDBfile(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestCreatesDBfile_")
	is.NoErr(err)
	// defer os.RemoveAll(dir)
	cfg := config.New().WithRoot(dir).WithDB("sled.db")

	t.Log("do")
	sl := sled.New(cfg)
	is.NotNil(sl)
	sl.Close()

	t.Log("check")
	if _, err = os.Stat(dir + "/sled.db"); os.IsNotExist(err) {
		is.Fail("DB not created " + dir + "/sled.db")
	}
}

// TODO: update test for new db path semantics.
func TestLateOpen(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestLateOpen_")
	is.NoErr(err)
	defer os.RemoveAll(dir)
	cfg := config.New().WithRoot(dir)

	t.Log("do")
	sl := sled.New(cfg)
	is.NotNil(sl)
	sl.Open(dir + "/sled.db")
	sl.Close()

	t.Log("check")
	if _, err = os.Stat(dir + "/sled.db"); os.IsNotExist(err) {
		is.Fail("DB not created " + dir + "/sled.db")
	}
}

func TestPersistance(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	b := make([]byte, 1024)
	n, err := rand.Read(b)
	is.NoErr(err)
	is.Equal(n, len(b))
	id, err := asset.Identify(bytes.NewReader(b))
	is.NoErr(err)
	t.Logf("data id: %s", id)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestPersistance_")
	is.NoErr(err)
	defer os.RemoveAll(dir)

	t.Log("do")
	t.Log(dir)
	cfg := config.New().WithRoot(dir).WithDB("sled.db")

	// #1
	sl := sled.New(cfg)
	is.NoErr(err)
	sl.Set("foo", "bar")

	sl.Set("bat", b)
	foo, err := sl.Get("foo")
	is.NoErr(err)
	bat, err := sl.Get("bat")
	is.NoErr(err)
	sl.Close()

	// #2
	sl2 := sled.New(cfg)
	defer sl2.Close()
	foo2, err := sl2.Get("foo")
	is.NoErr(err)
	bat2, err := sl2.Get("bat")
	is.NoErr(err)

	t.Log("check")
	is.Equal(foo.(string), "bar")
	is.Equal(bat.([]byte), b)
	is.Equal(foo2.(string), "bar")
	is.Equal(bat2.([]byte), b)
}

func TestIterator(t *testing.T) {
	t.Log("init")
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_TestIterator_")
	is.NoErr(err)
	cfg := config.New().WithRoot(dir).WithDB("db.sled")
	sl := sled.New(cfg)
	key_list := map[string]int{}
	rounds := 5
	i := 0

	t.Log("do")
	for i = 0; i < rounds; i++ {
		key := fmt.Sprintf("%08d", i)
		key_list[key] = i
		b, err := json.Marshal(i)
		is.NoErr(err)
		sl.Set(key, string(b))
		t.Log("key: ", key, ", b: ", string(b))
	}

	t.Log("check")
	for ele := range sl.Iterator(nil) {
		num, err := strconv.Atoi(ele.Value().(string))
		is.NoErr(err)
		t.Log("ele.Key: ", ele.Key(), ", ele.Value: ", num)
		is.Equal(key_list[ele.Key()], num)
	}
}
