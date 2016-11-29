package sled_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

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
	time.Sleep(time.Millisecond)
	value := sl2.Get("foo")
	is.NotNil(value)
	is.Equal(value.(string), "bar")
	sl2.Close()
}

// func TestItterate(t *testing.T) {
// 	is := is.New(t)
// 	sl := sled.New()

// 	keys := []string{
// 		"key1",
// 		"key2",
// 	}
// 	values := []string{
// 		"value 1",
// 		"value 42. Oh no, not again.",
// 	}

// 	err := sl.Set("bucket", keys[0], values[0])
// 	is.NoErr(err)

// 	err = sl.Set("bucket", keys[1], values[1])
// 	is.NoErr(err)

// 	i := 0
// 	sl.Iterate(func(k []byte, v []byte) bool {
// 		is.Equal(k, keys[i])
// 		is.Equal(v, values[i])
// 		i++
// 		return true
// 	})
// }

func TestIterator(t *testing.T) {
	is := is.New(t)
	sl := sled.New()

	key_list := map[string]int{}
	var i int
	for i = 0; i < 1000; i++ {
		key := fmt.Sprintf("%08d", i)
		key_list[key] = i
		b, err := json.Marshal(i)
		is.NoErr(err)
		sl.Set(key, b)
		t.Log("key: ", key, ", b: ", b)
		is.NoErr(err)
	}
	for ele := range sl.Iterator(nil) {
		num, err := strconv.Atoi(string(ele.Value().([]byte)))
		is.NoErr(err)
		t.Log("ele.Key: ", ele.Key(), ", ele.Value: ", num)
		is.Equal(key_list[ele.Key()], num)
	}
}
