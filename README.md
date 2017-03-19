# Sled 
[![GoDoc](https://godoc.org/github.com/Avalanche-io/sled?status.svg)](https://godoc.org/github.com/Avalanche-io/sled)
[![Go Report Card](https://goreportcard.com/badge/github.com/Avalanche-io/sled)](https://goreportcard.com/report/github.com/Avalanche-io/sled)
[![Build Status](https://travis-ci.org/Avalanche-io/sled.svg)](https://travis-ci.org/Avalanche-io/sled)
[![Coverage Status](https://coveralls.io/repos/github/Avalanche-io/sled/badge.svg)](https://coveralls.io/github/Avalanche-io/sled)

Sled is a high performance Key/Value store based on a [ctrie][1] data structure.  Sled is non-blocking and thread safe, meaning it is safe to access from any number of threads simultaneously.

Any type of data can be stored in a key, but sled maintains the benefits of the Go type system by enforcing the type. Values must be accessed by passing an empty type value of the same type into the Get method.

[1]: ctries_paper.pdf

## Usage

`go get "github.com/Avalanche-io/sled"`

Create a sled with sled.New().

`sl := sled.New()`

Setting a key. Sled accepts any type.

```go
sl.Set("key", "value")
sl.Set("Answer to the Ultimate Question", 42)
sl.Set("primes", []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29})
```

Getting a value. 

```go
var primes []int
var ultimate_answer int
var right_value_type string
var wrong_value_type []byte

err := sl.Get("Answer to the Ultimate Question", &ultimate_answer)
fmt.Printf("Answer to the Ultimate Question: %d, err: %t\n", ultimate_answer, err != nil)
err = sl.Get("primes", &primes)
fmt.Printf("Primes: %v, err: %t\n", primes, err != nil)
err = sl.Get("key", &wrong_value_type)
fmt.Printf("key: %v, err: %t\n", right_value_type, err != nil)
fmt.Printf("key (wrong type): %v, err: %s\n", wrong_value_type, err)
err = sl.Get("key", &right_value_type)
```

Setting a key conditionally.  SetIfNil will only assign the value if the key is not already set.

`SetIfNil(string, interface{}) bool`

Deleting a key.

`Delete(string) (interface{}, bool)`

Close, when done with a sled close it, to free resources.

`Close() error`

Iterating over all keys, can be done with range expression.

```go
stop := chan struct{} // or nil if interruption isn't needed.
for elm := range sl.Iterate(stop) {
    fmt.Printf("key: %s  value: %v\n", elm.Key(), elm.Value())
    elm.Close() // close the Element when done 
}
```

A Snapshot is a nearly zero cost copy of a sled that will not be effected by future changes to the source sled. It can be made mutable or immutable by setting the argument to `sled.ReadWrite`, or `sled.ReadOnly`.

```go
sl_mutable := sl.Snapshot(sled.ReadWrite)
sl_immutable := sl.Snapshot(sled.ReadOnly)
```

## Example

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
