.PHONY: build-all clean swagger release mockery
BIN = $(addprefix bin/,$(shell ls cmd))
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE := $(shell date)

build-all: $(BIN)

bin/%: cmd/%/main.go
	GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-X 'github.com/sapcc/archer/internal/config.Version=$(COMMIT)' -X 'github.com/sapcc/archer/internal/config.BuildTime=$(DATE)'" -o $@ $<

mockery:
	mockery

release:
	GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'github.com/sapcc/archer/internal/config.Version=$(COMMIT)' -X 'github.com/sapcc/archer/internal/config.BuildTime=$(DATE)'" -o bin/archerctl_darwin_adm64 cmd/archerctl/main.go
	GOOS=darwin GOARCH=arm64 go build -ldflags="-X 'github.com/sapcc/archer/internal/config.Version=$(COMMIT)' -X 'github.com/sapcc/archer/internal/config.BuildTime=$(DATE)'" -o bin/archerctl_darwin_arm64 cmd/archerctl/main.go
	GOOS=windows GOARCH=amd64 go build -ldflags="-X 'github.com/sapcc/archer/internal/config.Version=$(COMMIT)' -X 'github.com/sapcc/archer/internal/config.BuildTime=$(DATE)'" -o bin/archerctl.exe cmd/archerctl/main.go
	GOOS=linux GOARCH=amd64 go build -ldflags="-X 'github.com/sapcc/archer/internal/config.Version=$(COMMIT)' -X 'github.com/sapcc/archer/internal/config.BuildTime=$(DATE)'" -o bin/archerctl_linux_x86_64 cmd/archerctl/main.go


swagger:
	swagger generate server --exclude-main --copyright-file COPYRIGHT.txt
	swagger generate model --copyright-file COPYRIGHT.txt
	swagger generate client --copyright-file COPYRIGHT.txt

markdown:
	swagger generate markdown --copyright-file COPYRIGHT.txt --output= docs/api.md

clean:
	rm -f bin/*
