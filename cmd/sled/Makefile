.phony: all clean

all: sled

sled: main.go
	go build -ldflags "-X main.GitId=`git rev-parse --short=7 HEAD`"

clean: 
	rm sled