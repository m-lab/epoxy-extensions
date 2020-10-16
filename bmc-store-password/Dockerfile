FROM golang:1.14-alpine3.12
ADD . /go/src/github.com/m-lab/epoxy-extensions/bmc-store-password
RUN go get -v github.com/m-lab/epoxy-extensions/bmc-store-password
ENTRYPOINT ["/go/bin/bmc-store-password"]
