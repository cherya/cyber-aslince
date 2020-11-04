GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOINSTALL=$(GOCMD) install

BINARY_PATH=./bin
BINARY_NAME=cyberaslince
MAIN_NAME=./cmd/cyber-aslince

PROJECT_PATH=$(shell pwd)
GOBIN_PATH=$(GOPATH)/bin

all: test build

build:
	export CGO_CFLAGS="$(pkg-config --cflags MagickWand)"
	export CGO_LDFLAGS="$(pkg-config --libs MagickWand)"
	export CGO_CFLAGS_ALLOW='-Xpreprocessor'
	$(GOBUILD) -o $(BINARY_PATH)/$(BINARY_NAME) -v $(MAIN_NAME)

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: build
	$(BINARY_PATH)/$(BINARY_NAME)
