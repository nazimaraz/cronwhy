BIN := cronwhy

.PHONY: build test lint fuzz golden clean

build:
	go build -o $(BIN) ./cmd/cronwhy

test:
	go test ./... -race

lint:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "gofmt: files above need formatting" && exit 1)
	go vet ./...

fuzz:
	go test ./internal/cronexpr -run='^$$' -fuzz=FuzzParse -fuzztime=30s

golden:
	go test ./cmd/cronwhy -update

clean:
	rm -f $(BIN)
