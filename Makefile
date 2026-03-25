BINARY := codesnippetd
GO := go

.PHONY: all clean test

all: $(BINARY)

SRCS := $(wildcard *.go)

$(BINARY): $(SRCS) go.mod go.sum
	$(GO) build -o $(BINARY) .

test:
	$(GO) test ./...

clean:
	rm -f $(BINARY)
