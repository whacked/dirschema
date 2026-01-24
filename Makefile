BINARY := dirschema

.PHONY: build
build:
	go build -o $(BINARY) ./cmd/dirschema
