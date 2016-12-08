# Sled 
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](./LICENSE)
[![GoDoc](https://godoc.org/github.com/Avalanche-io/sled?status.svg)](https://godoc.org/github.com/Avalanche-io/sled)
[![Go Report Card](https://goreportcard.com/badge/github.com/Avalanche-io/sled)](https://goreportcard.com/report/github.com/Avalanche-io/sled)
[![Stories in Ready](https://badge.waffle.io/Avalanche-io/sled.png?label=ready&title=Ready)](https://waffle.io/Avalanche-io/sled)
[![Build Status](https://travis-ci.org/Avalanche-io/sled.svg?branch=master)](https://travis-ci.org/Avalanche-io/sled)
[![Coverage Status](https://coveralls.io/repos/github/Avalanche-io/sled/badge.svg?branch=master)](https://coveralls.io/github/Avalanche-io/sled?branch=master)


Sled is a very high performance thread safe Key/Value store based on a _ctrie_ data structure with automatic persistence to disk via non-blocking snapshots.

## Features

- Multi-thread safe
- Non-blocking
- Optional database storage
- Optional event notifications

More to come: Versions, TTL (time to live), performance improvements, and benchmarks.

## Example Usage

```go
import(
    "fmt"

    "github.com/Avalanche-io/sled"
    "github.com/Avalanche-io/sled/config"
)

func main() {
    // Add database storage to the configuration,
    // in the default location ($HOME/.sled/sled.db)
    cfg := config.New().WithDB("sled.db")
    sl := sled.New(cfg)
    defer sl.Close()

    key := "forty two"
    sl.Set(key, 42)

    v, err := sl.Get("forty two")

    if err != nil {
        panic(err)
    }

    fmt.Printf("key: %s, \tvalue: %v, \ttype: %T\n", key, v, v)

}

```