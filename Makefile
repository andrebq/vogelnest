.PHONY: watch build dist

watch:
	modd

build:
	go build ./...


dist:
	go build .

run:
	glua run.lua
