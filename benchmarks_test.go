package sled_test

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"github.com/Avalanche-io/sled"
)

func BenchmarkMapSet(b *testing.B) {
	m := make(map[string]interface{})
	for n := 0; n < b.N; n++ {
		m[strconv.Itoa(n)] = n
	}
}

func BenchmarkMapSetGet(b *testing.B) {
	m := make(map[string]interface{})
	for n := 0; n < b.N; n++ {
		m[strconv.Itoa(n)] = n
	}
	var v interface{}
	for n := 0; n < b.N; n++ {
		v = m[strconv.Itoa(n)]
		if v.(int) != n {
			b.Fail()
		}
	}
}

func BenchmarkMapSetParallel(b *testing.B) {
	m := make(map[string]interface{})
	mutex := sync.Mutex{}
	b.RunParallel(func(pb *testing.PB) {
		n := rand.Int()
		for pb.Next() {
			mutex.Lock()
			m[strconv.Itoa(n)] = n
			mutex.Unlock()
			n++
		}
	})
}

func BenchmarkMapSetGetParallel(b *testing.B) {
	m := make(map[string]interface{})
	mutex := sync.Mutex{}
	b.RunParallel(func(pb *testing.PB) {
		n := rand.Int()
		var v interface{}
		for pb.Next() {
			mutex.Lock()
			m[strconv.Itoa(n)] = n
			v = m[strconv.Itoa(n)]
			if v.(int) != n {
				b.Fail()
			}
			mutex.Unlock()
			n++
		}
	})
}

func BenchmarkSledSet(b *testing.B) {
	sl := sled.New()
	for n := 0; n < b.N; n++ {
		sl.Set(strconv.Itoa(n), n)
	}
}

func BenchmarkSledSetGet(b *testing.B) {
	sl := sled.New()
	for n := 0; n < b.N; n++ {
		sl.Set(strconv.Itoa(n), n)
	}
	var v int
	for n := 0; n < b.N; n++ {
		_ = sl.Get(strconv.Itoa(n), v)
		if v != n {
			b.Fail()
		}
	}
}

func BenchmarkSledSetParallel(b *testing.B) {
	sl := sled.New()
	b.RunParallel(func(pb *testing.PB) {
		n := rand.Int()
		for pb.Next() {
			sl.Set(strconv.Itoa(n), n)
			n++
		}
	})
}

func BenchmarkSledSetGetParallel(b *testing.B) {
	sl := sled.New()
	b.RunParallel(func(pb *testing.PB) {
		n := rand.Int()
		var v int
		for pb.Next() {
			sl.Set(strconv.Itoa(n), n)

			_ = sl.Get(strconv.Itoa(n), v)
			if v != n {
				b.Fail()
			}
			n++
		}
	})
}
