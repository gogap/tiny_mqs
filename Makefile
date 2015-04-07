.PHONY: all linux local

COMPONENT_DIR=$(notdir $(PWD))
EXEC_NAME=$(notdir $(PWD))
COMPONENTS_ROOT=github.com/gogap
IMAGE=google/golang

all: linux

linux:
	docker run --rm -e GOPATH=/gopath -v $(PWD):/out  -v $(GOPATH)/src:/gopath/src -w /gopath/src/$(COMPONENTS_ROOT)/$(COMPONENT_DIR) -e GOOS=linux -e GOARCH=amd64 $(IMAGE) go build -v -o /out/$(EXEC_NAME)

local:
	go build -v -o ./$(EXEC_NAME) $(COMPONENTS_ROOT)/$(COMPONENT_DIR)