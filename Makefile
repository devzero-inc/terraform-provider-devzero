default: fmt lint generate install

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# files to be copied
SERVICES_DIR ?= ../services

# source dir for the generated proto bits, and connect RPC bits
SOURCE_PROTO_DIR = dakr/proto/api/v1
SOURCE_GEN_PB_DIR = dakr/gen/api/v1
SOURCE_GEN_CONNECT_DIR = dakr/gen/api/v1/apiv1connect

# target dir for the generated proto bits, and connect RPC bits
TARGET_PROTO_DIR = internal/proto/api/v1
TARGET_GEN_PB_DIR = internal/gen/api/v1
TARGET_GEN_CONNECT_DIR = internal/gen/api/v1/apiv1connect

PROTO_FILES = common.proto k8s.proto recommendation.proto
GEN_PB_FILES = common.pb.go k8s.pb.go recommendation.pb.go
GEN_CONNECT_FILES = k8s.connect.go recommendation.connect.go

.PHONY: proto
proto:
	@echo "Copying protos from $(SERVICES_DIR)/$(SOURCE_PROTO_DIR) to $(TARGET_PROTO_DIR)"
	@for proto in $(PROTO_FILES); do \
		if [ -f $(SERVICES_DIR)/$(SOURCE_PROTO_DIR)/$$proto ]; then \
			cp $(SERVICES_DIR)/$(SOURCE_PROTO_DIR)/$$proto $(TARGET_PROTO_DIR)/; \
		else \
			echo "Error: Missing file: $$proto"; \
			exit 1; \
		fi; \
	done
	@echo "Done copying protos."

	@echo "Copying generated pb.go files from $(SERVICES_DIR)/$(SOURCE_GEN_PB_DIR) to $(TARGET_GEN_PB_DIR)"
	@for proto in $(GEN_PB_FILES); do \
		if [ -f $(SERVICES_DIR)/$(SOURCE_GEN_PB_DIR)/$$proto ]; then \
			cp $(SERVICES_DIR)/$(SOURCE_GEN_PB_DIR)/$$proto $(TARGET_GEN_PB_DIR)/; \
		else \
			echo "Error: Missing file: $$proto"; \
			exit 1; \
		fi; \
	done
	@echo "Done copying generated pb.go files."
	
	@echo "Copying generated pb.go files for connect rpc from $(SERVICES_DIR)/$(SOURCE_GEN_CONNECT_DIR) to $(TARGET_GEN_CONNECT_DIR)"
	@for proto in $(GEN_CONNECT_FILES); do \
		if [ -f $(SERVICES_DIR)/$(SOURCE_GEN_CONNECT_DIR)/$$proto ]; then \
			cp $(SERVICES_DIR)/$(SOURCE_GEN_CONNECT_DIR)/$$proto $(TARGET_GEN_CONNECT_DIR)/; \
		else \
			echo "Error: Missing file: $$proto"; \
			exit 1; \
		fi; \
	done
	@echo "Done copying generated pb.go files."

	@echo "Replacing import paths in copied .go files..."
	@OLD_IMPORT="github.com/devzero-inc/services/dakr/gen/api/v1"; \
	NEW_IMPORT="github.com/devzero-inc/terraform-provider-devzero/internal/gen/api/v1"; \
	for f in $(TARGET_GEN_PB_DIR)/*.go $(TARGET_GEN_CONNECT_DIR)/*.go; do \
		if [ "$$(uname)" = "Darwin" ]; then \
			sed -i '' "s|$$OLD_IMPORT|$$NEW_IMPORT|g" $$f; \
		else \
			sed -i "s|$$OLD_IMPORT|$$NEW_IMPORT|g" $$f; \
		fi; \
	done
	@echo "Import path replacement complete."

.PHONY: fmt lint test testacc build install generate

