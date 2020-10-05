.PHONY: watch build dist run build-image build-ui

IMAGE_NAME?=andrebq/vogelnest
IMAGE_TAG?=latest
IMAGE_FULL_NAME=$(IMAGE_NAME):$(IMAGE_TAG)

deps:
	go get -u github.com/aerogo/pack/...

watch:
	modd

build:
	go build ./...
	go test ./internal/lib/trail
	go install ./cmd/vogelctl

build-image:
	docker build -t $(IMAGE_FULL_NAME) .

push:
	docker push $(IMAGE_FULL_NAME)

run:
	glua run.lua
