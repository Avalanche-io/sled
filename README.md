# Sled 
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![GoDoc](https://godoc.org/github.com/Avalanche-io/sled?status.svg)](https://godoc.org/github.com/Avalanche-io/sled)
[![Go Report Card](https://goreportcard.com/badge/github.com/Avalanche-io/sled)](https://goreportcard.com/report/github.com/Avalanche-io/sled)
[![Stories in Ready](https://badge.waffle.io/Avalanche-io/sled.png?label=ready&title=Ready)](https://waffle.io/Avalanche-io/sled)
[![Build Status](https://travis-ci.org/Avalanche-io/sled.svg?branch=master)](https://travis-ci.org/Avalanche-io/sled)
[![Coverage Status](https://coveralls.io/repos/github/Avalanche-io/sled/badge.svg?branch=master)](https://coveralls.io/github/Avalanche-io/sled?branch=master)

Sled is a very high performance thread safe Key/Value store based on a [ctrie][1] data structure.

[1]: https://axel22.github.io/resources/docs/ctries-snapshot.pdf

## Motivation

Sled is more than twice as fast as a go `map` when multi-threaded.

Sled is thread safe, and non-blocking, while Go's built in `map` is not thread safe, so it must be protected with blocking thread synchronization (i.e. mutex lock, channel, etc.)

## Features

- Thread safe
- Non-blocking
- Zero cost Snapshots
- Iterator
- [TODO] Optional concurrent save / load from database
- [TODO] Optional Read-Through caching

## CLI Example App

`sled key [value] [key value ...]`

Sled will save data in a `sled.db` file, in the local directory.  

To set keys simply provide pairs of arguments. `sled` interprets each pair of arguments as `key value`.  Providing a single argument will cause `sled` to return the value (if any) for that key.  

## Benchmarks

```
BenchmarkMapSet-24                   1000000          1021 ns/op
BenchmarkMapSetGet-24                1000000          1401 ns/op
BenchmarkMapSetParallel-24           1000000          1335 ns/op
BenchmarkMapSetGetParallel-24        1000000          1527 ns/op
BenchmarkSledSet-24                  1000000          1522 ns/op
BenchmarkSledSetGet-24               1000000          2528 ns/op
BenchmarkSledSetParallel-24          3000000           566 ns/op
BenchmarkSledSetGetParallel-24       2000000           683 ns/op
```

Sled is slower than map on a small number of threads, but becomes much faster then map as the number of threads increase up to the hyperthread limit of the system.  Future work will improve Sled's performance for lower thread counts.

## Example Usage

```go
package main

import (
    "fmt"

    "github.com/Avalanche-io/sled"
)

func main() {
    sl := sled.New()

    key := "The meaning of life"
    sl.Set(key, 42)
    
    var v int
    err := sl.Get(key, &v)
    if err != nil {
        panic(err)
    }
    fmt.Printf("key: %s, \tvalue: %v\n", key, v)
}
```