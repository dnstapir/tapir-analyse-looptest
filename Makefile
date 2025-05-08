VERSION:=`cat ./VERSION`
COMMIT:=`git describe --dirty=+WiP --always 2> /dev/null || echo "no-vcs"`

.PHONY: test coverage fmt clean

all:
	go build -v -ldflags "-X 'main.version=$(VERSION)-$(COMMIT)'" -o ./bin/ ./cmd/...

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html="coverage.out"

fmt:
	go fmt ./...

clean:
	-rm -rf ./bin/
	-rm coverage.out
