# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
FROM golang:1.13.7-alpine3.11 as builder
WORKDIR /go/src/github.com/apulis/bmod/ai-lab-backend

ENV GOPROXY=https://goproxy.cn
ENV GO111MODULE=on
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk --no-cache add git pkgconfig build-base

# Cache go modules
COPY go.mod .
COPY go.sum .
ADD . .
RUN go mod download

#RUN swag init --parseDependency --parseInternal && GO111MODULE=${GO111MODULE} go build cmd/ai_lab.go -o /go/bin/ai_lab
RUN  GO111MODULE=${GO111MODULE} go build  -o /go/bin/ai_lab cmd/ai_lab.go

FROM alpine:3.11
RUN apk --no-cache add ca-certificates libdrm
WORKDIR /app/
COPY --from=0 /go/bin/ai_lab .
CMD ["./ai_lab"]
