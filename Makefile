# note: call scripts from /scripts

image_name ?=ai-lab
harbor_addr=harbor.apulis.cn:8443/aistudio/bmod/${image_name}
tag        =aistudio-v0.1.0
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
	go build -o ${image_name} cmd/${image_name}.go

docker:
	docker build -f build/Dockerfile . -t ${image_name}:${tag} --build-arg project=${image_name}
kind:docker
	kind load docker-image ${image_name}:${tag} --name river
gen-swagger:
	swag init -g cmd/${image_name}.go -o api

dist: testarch
	docker build -t ${image_name} .
	docker tag ${image_name} ${harbor_addr}/${arch}:${tag}
	docker push ${harbor_addr}/${arch}:${tag}
manifest:
	./docker_manifest.sh ${harbor_addr}:${tag}
