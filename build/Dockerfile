#/* ******************************************************************************
#* 2019 - present Contributed by Apulis Technology (Shenzhen) Co. LTD
#*
#* This program and the accompanying materials are made available under the
#* terms of the MIT License, which is available at
#* https://www.opensource.org/licenses/MIT
#*
#* See the NOTICE file distributed with this work for additional
#* information regarding copyright ownership.
#* Unless required by applicable law or agreed to in writing, software
#* distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
#* WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
#* License for the specific language governing permissions and limitations
#* under the License.
#*
#* SPDX-License-Identifier: MIT
#******************************************************************************/
FROM golang:alpine

ARG project=ai-lab

WORKDIR /go/src/github.com/apulis/bmod/$project


ENV GOPROXY=https://goproxy.cn
ENV GO111MODULE=on
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk --no-cache add git pkgconfig build-base

# Cache go modules
ADD go.mod .
ADD go.sum .
ADD deps deps
RUN go mod download
ADD . .


#RUN swag init --parseDependency --parseInternal && GO111MODULE=${GO111MODULE} go build cmd/ai_lab.go -o /go/bin/ai_lab
RUN  GO111MODULE=${GO111MODULE} go build -buildmode=pie  -o /go/bin/$project cmd/$project.go
RUN  strip /go/bin/$project


FROM alpine:3.11
ARG project=ai-lab
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk --no-cache add ca-certificates libdrm
COPY --from=0 /usr/local/go/lib/time/zoneinfo.zip /opt/zoneinfo.zip
ENV ZONEINFO /opt/zoneinfo.zip
ENV PROJECT_NAME=$project
WORKDIR /app/
COPY --from=0 /go/bin/$project .

CMD ./${PROJECT_NAME}
