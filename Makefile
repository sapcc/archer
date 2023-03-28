.PHONY: build-all clean swagger
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
