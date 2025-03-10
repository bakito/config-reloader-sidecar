FROM golang:1.24-alpine AS builder

RUN apk update && apk add upx

WORKDIR /workspace

ADD . .
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o config-reloader-sidecar . && \
    upx --best --lzma config-reloader-sidecar

# Runtime

FROM gcr.io/distroless/static-debian11:latest

COPY --from=builder /workspace/config-reloader-sidecar .

CMD ["/config-reloader-sidecar"]
