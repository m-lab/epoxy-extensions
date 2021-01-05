FROM golang:1.15-alpine3.12
ADD . /go/src/github.com/m-lab/epoxy-extensions/bmc_store_password
RUN apk add --no-cache git
RUN go get -v github.com/m-lab/epoxy-extensions/bmc_store_password
ENTRYPOINT ["/go/bin/bmc_store_password"]
