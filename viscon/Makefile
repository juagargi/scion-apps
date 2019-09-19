
all: example


capi: scion_api.go
	go build -o scion_api.so -buildmode=c-shared scion_api.go

example: capi example.py
	python3 example.py
