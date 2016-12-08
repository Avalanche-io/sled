package main

import (
	"fmt"
	"os"

	"github.com/Avalanche-io/sled"
	"github.com/Avalanche-io/sled/config"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
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

	if len(args) > 2 || len(args) == 0 {
		logger.Fatalf("Usage:\tsled key value\n")
		return -1
	}

	sl := sled.New(config.New().WithDB("sled.db"))
	defer sl.Close()

	if len(args) == 2 {
		sl.Set(args[0], args[1])
	}

	if len(args) == 1 {
		v, err := sl.Get(args[0])
		if err != nil {
			logger.WithError(err)
			return -1
		}
		fmt.Println(v)
	}

	return 0
}
