ifndef GO
GO := go
endif

all: deps test build

deps:
	$(GO) get ./...

build:
	$(GO) build -o bin/logdrain main.go

test:
	$(GO) test -v ./...

clean:
	rm bin/logdrain

install: build
	cp bin/logdrain ~/.local/bin
