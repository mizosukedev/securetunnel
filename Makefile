SHELL = /bin/bash

# tools
PROTOBUF      := $(shell command -v protoc)
PROTOC_GEN_GO := $(shell command -v protoc-gen-go)

# AWS secure tunneling message file.
SECURE_TUNNEL_VERSION := v2.1.0
PROTO_FILE_URL        := https://raw.githubusercontent.com/aws-samples/aws-iot-securetunneling-localproxy/$(SECURE_TUNNEL_VERSION)/resources/Message.proto
PROTO_FILE_DIR        := $(CURDIR)/proto
PROTO_FILE_NAME       := Message.proto
PROTO_GO_PACKAGE      := github.com/mizosukedev/securetunnel/client
PROTO_FILE_PATH       := $(PROTO_FILE_DIR)/$(PROTO_FILE_NAME)
MESSAGE_FILE_`OUT_DIR  := $(CURDIR)/client

.PHONY: \
	install-tools \
	update-proto  \
	clean         \

install-tools:
ifndef PROTOBUF
	@echo You need to install protobuf compiler.
	@echo ex. \"sudo apt install protobuf-compiler\"
	exit 1
endif
ifndef PROTOC_GEN_GO
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
endif

$(PROTO_FILE_DIR):
	mkdir -p $(PROTO_FILE_DIR)

$(PROTO_FILE_PATH): $(PROTO_FILE_DIR)
	curl -L -o "$(PROTO_FILE_PATH)" "$(PROTO_FILE_URL)"

# See https://developers.google.com/protocol-buffers/docs/reference/go-generated
update-proto: install-tools $(PROTO_FILE_PATH)
	protoc \
		-I "$(PROTO_FILE_DIR)"                             \
		--go_opt=paths=source_relative                     \
		--go_opt=M"$(PROTO_FILE_NAME)"=$(PROTO_GO_PACKAGE) \
		--go_out="$(MESSAGE_FILE_`OUT_DIR)"                 \
		"$(PROTO_FILE_NAME)"

clean:
	rm -f $(PROTO_FILE_PATH)