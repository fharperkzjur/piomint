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

docker:
	docker build -f Dockerfile . -t ai-lab:v0.1.0

dist: docker
	docker tag ai-lab:v0.1.0 harbor.apulis.cn:8443/release/apulistech/ai-lab:v0.1.0
	docker push harbor.apulis.cn:8443/release/apulistech/ai-lab:v0.1.0
gen-swagger:
	swag init -g cmd/ai_lab.go -o api
