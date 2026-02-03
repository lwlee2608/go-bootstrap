GO = $(shell which go 2>/dev/null)

APP				:= genesis
VERSION 		?= v0.1.3
LDFLAGS 		:= -ldflags "-X main.AppVersion=$(VERSION)"

.PHONY: all build clean run test install

all: clean build

clean:
	$(GO) clean -testcache
	$(RM) -rf bin/*
build:
	$(GO) build -o bin/$(APP)	$(LDFLAGS) cmd/$(APP)/*.go
run:
	$(GO) run $(LDFLAGS) cmd/$(APP)/*.go
test:
	$(GO) test -v ./...
install: build
	mkdir -p ~/.local/bin
	cp bin/$(APP) ~/.local/bin/$(APP)
