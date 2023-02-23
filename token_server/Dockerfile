FROM golang:1.19
ADD . /go/src/github.com/m-lab/epoxy-extensions/token_server
WORKDIR /go/src/github.com/m-lab/epoxy-extensions/token_server
RUN go install -v .
ENTRYPOINT ["/go/bin/token_server"]

