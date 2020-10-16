FROM golang:1.14-alpine3.12 as extension-build
RUN apk add --no-cache git
ADD . /go/src/github.com/m-lab/epoxy-extensions/bmc-store-password
RUN cd /go/src/github.com/m-lab/epoxy-extensions/bmc-store-password && ./build.sh

# Now copy the built image into the minimal base image
FROM alpine:3.12
COPY --from=ext-build /go/bin/bmc-store-password /
WORKDIR /
ENTRYPOINT ["/bmc-store-password"]
