FROM golang:1.10
ADD . /go/src/github.com/m-lab/epoxy-extensions/token_server
RUN go get -v github.com/m-lab/epoxy-extensions/token_server
ENTRYPOINT ["/go/bin/token_server"]
