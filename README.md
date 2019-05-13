# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=kube-compose
SYSTEMS=darwin linux windows

all: build tests

build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

tests: 
	$(GOTEST) -v ./...

clean: 
	$(GOCLEAN)
	rm -rf release

run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

deps:
	$(GOGET) github.com/stretchr/testify
	$(GOGET) github.com/urfave/cli

modules:
	$(GOMOD) tidy
	${GOMOD} download


releases:
	$(foreach SYSTEM, $(SYSTEMS), \
	mkdir -p release/$(SYSTEM); \
	CGO_ENABLED=0 GOOS=$(SYSTEM) GOARCH=amd64 $(GOBUILD) -o release/$(SYSTEM)/$(BINARY_NAME).$(SYSTEM); \
	cd release/$(SYSTEM)/; \
	tar -zcvf $(BINARY_NAME).$(SYSTEM).tar.gz $(BINARY_NAME).$(SYSTEM); \
	cd ../../; \
	)
