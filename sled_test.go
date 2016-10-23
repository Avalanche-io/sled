package sled_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/cheekybits/is"

	"github.com/Avalanche-io/sled"
)

func TestCreateItem(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	is.NotNil(sl)
}

func TestGetSetKey(t *testing.T) {
	is := is.New(t)

	sl := sled.New()

	sl.Set("foo", "bar")
	value := sl.Get("foo")
	is.Equal(value.(string), "bar")
}

func TestCreatesDB(t *testing.T) {
	is := is.New(t)
	dir, err := ioutil.TempDir("/tmp", "sledTest_")
	is.NoErr(err)

	defer os.RemoveAll(dir)

	db_path := dir + "/sled.db"
	sl, err := sled.Open(db_path)
	is.NoErr(err)
	sl.Close()
	if _, err = os.Stat(db_path); os.IsNotExist(err) {
		is.Fail("DB not created " + db_path)
	}
}

func TestLateOpen(t *testing.T) {
	is := is.New(t)
	ch := temp_dir(is)
	db_path := <-ch
	defer close(ch)

	sl := sled.New()
	is.NotNil(sl)
	err := sl.Open(db_path)
	is.NoErr(err)
	sl.Close()
	if _, err = os.Stat(db_path); os.IsNotExist(err) {
		is.Fail("DB not created " + db_path)
	}
}

func temp_dir(is is.I) chan string {
	dir, err := ioutil.TempDir("/tmp", "sledTest_")
	is.NoErr(err)
	ch := make(chan string)

	go func() {
		ch <- dir + "/sled.db"
		<-ch // wait for channel to be closed
		os.RemoveAll(dir)
	}()
	return ch
}

func TestPersistance(t *testing.T) {
	is := is.New(t)
	ch := temp_dir(is)
	db_path := <-ch
	defer close(ch)

	sl, err := sled.Open(db_path)
	is.NoErr(err)
	sl.Set("foo", "bar")
	sl.Close()

	sl2, err := sled.Open(db_path)
	is.NoErr(err)
	value := sl2.Get("foo")
	is.NotNil(value)
	is.Equal(value.(string), "bar")
	sl2.Close()
}
