FROM golang:1.18
ADD . /go/src/github.com/m-lab/epoxy-extensions
WORKDIR /go/src/github.com/m-lab/epoxy-extensions
RUN go build -o server .
RUN mv server /usr/local/bin
ENTRYPOINT ["/usr/local/bin/server"]

