.PHONY: all example

all: capi example sink


capi: scion_api.so

scion_api.so: scion_api.go
	go build -o scion_api.so -buildmode=c-shared scion_api.go

example: capi example.py
	@echo To test the API, run \"python3 example.py\"

sink: sink.go
	go build -o sink sink.go
	./sink
