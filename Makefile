.PHONY: watch build dist run build-image build-ui

deps:
	go get -u github.com/aerogo/pack/...

watch:
	modd

build:
	go build ./...

build-image:
	docker build -t andrebq/vogelnest:latest .

dist:
	go build .

run:
	glua run.lua
