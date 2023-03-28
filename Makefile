.PHONY: build-all clean swagger
PROTOC_FILES = $(shell find . -type f -name '*.proto')
PB_FILES = $(patsubst %.proto, %.pb.go, $(PROTOC_FILES))
PB_MICRO_FILES = $(patsubst %.proto, %.pb.micro.go, $(PROTOC_FILES))
BIN = $(addprefix bin/,$(shell ls cmd))

build-all: $(BIN)

bin/%: cmd/%/main.go
	go build -o $@ $<

swagger:
	swagger generate server --exclude-main --copyright-file COPYRIGHT.txt
	swagger generate model --copyright-file COPYRIGHT.txt
	swagger generate client --copyright-file COPYRIGHT.txt

markdown:
	swagger generate markdown --copyright-file COPYRIGHT.txt --output= docs/api.md

clean:
	rm -f bin/*
