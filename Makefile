# note: call scripts from /scripts

image_name =ai-lab
harbor_addr=harbor.apulis.cn:8443/huawei630/apulistech/${image_name}
tag        =v0.1.0
arch       =$(shell arch)

testarch:
ifeq (${arch}, x86_64)
	@echo "current build host is amd64 ..."
	$(eval arch=amd64)
else ifeq (${arch},aarch64)
	@echo "current build host is arm64 ..."
	$(eval arch=arm64)
else
	echo "cannot judge host arch:${arch}"
	exit -1
endif
	@echo "arch type:$(arch)"





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

gen-swagger:
	swag init -g cmd/ai_lab.go -o api

dist: testarch
	docker build -t ${image_name} .
	docker tag ${image_name} ${harbor_addr}/${arch}:${tag}
	docker push ${harbor_addr}/${arch}:${tag}
manifest:
	./docker_manifest.sh ${harbor_addr}:${tag}
