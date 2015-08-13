V=$(shell git describe --tags)

all:
	go build -ldflags "-X main.version $(V)"

