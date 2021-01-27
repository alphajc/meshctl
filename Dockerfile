FROM golang:1.15.7-buster
ADD . /build
WORKDIR /build
ENV GOPROXY https://goproxy.io,direct
RUN go build -o meshctl cmd/meshctl/main.go

FROM ccr.ccs.tencentyun.com/lzwk/kustomize:3.8.7
LABEL maintainer="Jerry<jiajun.chen@tenclass.com>"
COPY --from=0 /build/meshctl /usr/local/bin
ENTRYPOINT meshctl
