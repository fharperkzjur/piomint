# note: call scripts from /scripts
get-deps:
	git submodule init
	git submodule update
	go mod vendor

vet-check-all: get-deps
	go vet ./...

gosec-check-all: get-deps
	gosec ./...

bin: get-deps
	go build -o ai_lab cmd/ai_lab.go

docker: get-deps
	docker build -f Dockerfile . -t ai-lab:0.1

gen-swagger:
	swag init -g cmd/ai_lab.go -o api
