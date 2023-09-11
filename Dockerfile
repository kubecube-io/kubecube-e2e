# Build the manager binary
FROM golang:1.20.7-alpine3.18 as builder

WORKDIR /workspace

# Copy the go source
COPY go.mod go.mod
COPY go.sum go.sum
COPY e2e/ e2e/
COPY vendor/ vendor/
COPY Makefile Makefile

# Build
RUN CGO_ENABLED=0 GOOS=linux GO111MODULE=on go test -mod=vendor -c -o cube.test ./e2e

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.18
WORKDIR /workspace
ENV TZ Asia/Shanghai
COPY --from=builder /workspace/cube.test .
COPY tomcat-10.3.10.tgz tomcat-10.3.10.tgz
CMD ["/workspace/cube.test", "-test.v"]
