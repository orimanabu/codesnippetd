BINARY := codesnippetd
GO := go

.PHONY: all clean test

all: $(BINARY)

$(BINARY):
	$(GO) build -o $(BINARY) .

test:
	$(GO) test ./...

clean:
	rm -f $(BINARY)
