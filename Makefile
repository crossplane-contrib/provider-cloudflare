# Set the shell to bash always
SHELL := /bin/bash

# Options
ORG_NAME=localhost:5432
PROVIDER_NAME=provider-cloudflare
CONTAINER_REGISTRY=cluster-registry.local

# Tools
CLUSTER=$(shell which k3d)
LINT=$(shell which golangci-lint)
LINT_CONTAINER_IMAGE=golangci/golangci-lint:v1.45.2

cluster: clean
	$(CLUSTER) cluster create --registry-create $(CONTAINER_REGISTRY):0.0.0.0:5432

clean:
	$(CLUSTER) cluster delete

build: install test
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./bin/$(PROVIDER_NAME)-controller cmd/provider/main.go

image: install test
	docker build . -t $(ORG_NAME)/$(PROVIDER_NAME):latest -f cluster/Dockerfile

image-push:
	docker push $(ORG_NAME)/$(PROVIDER_NAME):latest

run: install
	go run cmd/provider/main.go -d

all: image image-push install

install: generate
	kubectl apply -f package/crds/ -R

generate:
	go generate ./...
	@find package/crds -name *.yaml -exec sed -i.sed -e '1,2d' {} \;
	@find package/crds -name *.yaml.sed -delete

lint-local:
	docker run --rm -v $(shell pwd):/app -w /app $(LINT_CONTAINER_IMAGE) golangci-lint run -v

lint:
	$(LINT) run

tidy:
	go mod tidy

test:
	go test -v ./...

.PHONY: generate tidy lint lint-local clean cluster build image all run