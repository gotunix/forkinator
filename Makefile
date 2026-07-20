.PHONY: build test clean install uninstall

BINARY_NAME=forkinator
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin

build:
	go build -o $(BINARY_NAME) cmd/$(BINARY_NAME)/main.go

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)

install: build
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(BINARY_NAME) $(DESTDIR)$(BINDIR)/$(BINARY_NAME)

uninstall:
	rm -f $(DESTDIR)$(BINDIR)/$(BINARY_NAME)
