.PHONY: all
all: fmt doc build

.PHONY: build
build:
	go build .

.PHONY: doc
doc:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.19.2 generate -provider-name util

.PHONY: fmt
fmt:
	go fmt ./...
	terraform fmt
