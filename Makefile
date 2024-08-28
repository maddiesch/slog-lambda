.PHONY: test
test:
	go test -v -race ./...

.PHONY: benchmark
benchmark:
	go test -bench . -benchtime=10s ./... -benchmem
